import copy
from tx_interpreter.filters import (
    BalanceChangeFilter,
    TokenFilter,
    SignerFilter
)
from tx_interpreter.utils import (
    BalanceCalculator,
    DominantFigureFilter,
    BalanceUpdateLocator,
    get_currencies_involved
)
from tx_interpreter.functions.enricher.script import enricher

# Define the SOL mint address
SOL_MINT = "11111111111111111111111111111111"


def get_token_decimals(balance_updates, default=9):
    """
    Safely extract the `Decimals` field from a list of balance updates.
    Returns `default` if not found or invalid.
    """
    if not balance_updates or len(balance_updates) == 0:
        return default

    currency_info = balance_updates[0].get("Currency", {})
    decimals = currency_info.get("Decimals", default)
    try:
        decimals = int(decimals)
    except (TypeError, ValueError):
        decimals = default

    # Bound the decimals to avoid extreme or negative values
    if decimals < 0 or decimals > 18:
        decimals = default

    return decimals


def parse_amount(raw_amount, decimals):
    """
    Convert a raw integer-like amount to a float by dividing by 10^decimals.
    """
    return float(raw_amount) / (10 ** decimals)


def sol_to_xxx(data):
    """
    Processes a transaction to detect SOL-to-token or token-to-SOL swaps and enriches
    the data with additional details. All division-by-decimals is delegated to
    helper functions (parse_amount) and not hard-coded in this function.

    Args:
        data (dict): The transaction data containing balance updates, transaction details, etc.

    Returns:
        dict: A dictionary containing enriched swap details.
    """

    # 1) Make deep copies for partial filtering
    signer_data = copy.deepcopy(data)
    first_token_data = copy.deepcopy(data)
    sol_token_data = copy.deepcopy(data)

    # 2) Filter out only the signer's balance updates
    signer_data["BalanceUpdates"] = SignerFilter(signer_data).apply()

    # 3) Additional copies to isolate SOL vs. other token from the signer
    sol_token_signer_data = copy.deepcopy(signer_data)
    first_token_signer_data = copy.deepcopy(signer_data)

    # 4) Identify which tokens are involved
    tokens = get_currencies_involved(signer_data)

    if len(tokens) > 2:
        raise ValueError("[ERROR] Use xxx_to_xxx function for multi-token swaps.")

    if SOL_MINT not in tokens:
        raise ValueError("[ERROR] Use xxx_to_xxx function for non-SOL swaps.")

    sol_index = tokens.index(SOL_MINT)
    token_index = 1 - sol_index  # The "other token"

    # 5) Filter the original data so that each copy only has the relevant token's updates
    first_token_data["BalanceUpdates"] = TokenFilter(data, tokens[token_index]).apply()
    sol_token_data["BalanceUpdates"]   = TokenFilter(data, tokens[sol_index]).apply()

    # ...and similarly for the signer-specific data
    first_token_signer_data["BalanceUpdates"] = TokenFilter(signer_data, tokens[token_index]).apply()
    sol_token_signer_data["BalanceUpdates"]   = TokenFilter(signer_data, tokens[sol_index]).apply()

    # 6) Compute decimal-based differences for each token
    first_token_differences         = BalanceCalculator.calculate_balance_differences(first_token_data)
    sol_token_differences           = BalanceCalculator.calculate_balance_differences(sol_token_data)
    first_token_differences_signer  = BalanceCalculator.calculate_balance_differences(first_token_signer_data)
    sol_token_differences_signer    = BalanceCalculator.calculate_balance_differences(sol_token_signer_data)

    # 7) Detect a swap
    data["swapDetected"] = (
        (sum(sol_token_differences_signer) < 0  and sum(first_token_differences_signer) > 0)
        or (sum(sol_token_differences_signer) > 0  and sum(first_token_differences_signer) < 0)
    )

    # 8) Decide direction: If signer ends with more SOL, it’s token->SOL. Otherwise, SOL->token.
    # The existing code uses sum(sol_token_differences_signer) > 0 => "token->SOL".
    # We'll keep that logic for backward compatibility.
    sol_to_token = (sum(sol_token_differences_signer) > 0)

    # 9) Find the dominant figure for each token
    first_dom_figures = DominantFigureFilter.filter_dominant_figures(first_token_differences)
    sol_dom_figures   = DominantFigureFilter.filter_dominant_figures(sol_token_differences)

    # We only expect one "dominant figure" per token
    if len(first_dom_figures) > 1 or len(sol_dom_figures) > 1:
        raise ValueError("[ERROR] Failed to analyze swap.")

    # 10) Retrieve decimals for each side
    #     - For SOL, we know it's 9 decimals
    #     - For the "other token", glean from signer's updates
    first_token_decimals = get_token_decimals(first_token_signer_data["BalanceUpdates"], default=9)
    sol_decimals = 9

    # 11) Prepare variables that we'll populate
    swap_fees = 0.0

    if not sol_to_token:
        # -----------------------
        # SOL -> Token scenario
        # -----------------------
        from_token_symbol = "SOL"
        from_token_mint   = SOL_MINT
        to_token_symbol   = first_token_signer_data["BalanceUpdates"][0]["Currency"]["Symbol"]
        to_token_mint     = tokens[token_index]

        # The "dominant figures" are still raw integers from
        # BalanceCalculator, *unless* your BalanceCalculator also does parse_amount for you.
        # If you truly need to do it here, we do parse_amount:
        from_token_change = parse_amount(sol_dom_figures[0], sol_decimals)      # negative if user spent SOL
        to_token_change   = parse_amount(first_dom_figures[0], first_token_decimals)  # positive if user gained tokens

        # If you need a custom logic for fees, do it with parse_amount:
        # (original code used a manual /1_000_000_000)
        fee_part1 = parse_amount(min(sol_token_differences), sol_decimals)
        fee_part2 = parse_amount(sol_dom_figures[0], sol_decimals)
        swap_fees = abs(fee_part1 + fee_part2)

        # Get the biggest "to_token" update, presumably the user gained token
        to_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data["BalanceUpdates"],
            max(first_token_differences_signer)
        )
        # Convert pre/post to decimals
        to_token_pre_swap_bal  = parse_amount(to_token_update["BalanceUpdate"]["PreBalance"],  first_token_decimals)
        to_token_post_swap_bal = parse_amount(to_token_update["BalanceUpdate"]["PostBalance"], first_token_decimals)

        # For SOL, find the smallest difference (spent)
        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data["BalanceUpdates"],
            min(sol_token_differences_signer)
        )
        from_token_pre_swap_bal  = parse_amount(sol_token_update["BalanceUpdate"]["PreBalance"],  sol_decimals)
        from_token_post_swap_bal = parse_amount(sol_token_update["BalanceUpdate"]["PostBalance"], sol_decimals)

    else:
        # -----------------------
        # Token -> SOL scenario
        # -----------------------
        from_token_symbol = first_token_signer_data["BalanceUpdates"][0]["Currency"]["Symbol"]
        from_token_mint   = tokens[token_index]
        to_token_symbol   = "SOL"
        to_token_mint     = SOL_MINT

        from_token_change = parse_amount(first_dom_figures[0], first_token_decimals)  # negative if parted with token
        to_token_change   = parse_amount(sol_dom_figures[0],   sol_decimals)          # positive if gained SOL

        # Original code: swap_fees = abs(min(sol_token_differences) / 1_000_000_000)
        # We'll rely on parse_amount instead:
        fee_part = parse_amount(min(sol_token_differences), sol_decimals)
        swap_fees = abs(fee_part)

        # from_token update => the smallest difference is presumably the user parted with token
        from_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data["BalanceUpdates"],
            min(first_token_differences_signer)
        )
        from_token_pre_swap_bal  = parse_amount(from_token_update["BalanceUpdate"]["PreBalance"],  first_token_decimals)
        from_token_post_swap_bal = parse_amount(from_token_update["BalanceUpdate"]["PostBalance"], first_token_decimals)

        # For SOL, find the largest difference (gained)
        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data["BalanceUpdates"],
            max(sol_token_differences_signer)
        )
        to_token_pre_swap_bal  = parse_amount(sol_token_update["BalanceUpdate"]["PreBalance"],  sol_decimals)
        to_token_post_swap_bal = parse_amount(sol_token_update["BalanceUpdate"]["PostBalance"], sol_decimals)

    # 12) Build the final swap payload
    swap_data = {
        "BlockchainFees": {
            "Amount": data["Transaction"]["Fee"],
            "AmountUSD": None
        },
        "SwapFees": {
            "Amount": swap_fees,
            "AmountUSD": None
        },
        "FromToken": {
            "AmountChange": from_token_change,
            "AmountSOL": None,
            "AmountUSD": None,
            "Mint": from_token_mint,
            "PostSwapBalance": from_token_post_swap_bal,
            "PostSwapBalanceSOL": None,
            "PostSwapBalanceUSD": None,
            "PreSwapBalance": from_token_pre_swap_bal,
            "PreSwapBalanceSOL": None,
            "PreSwapBalanceUSD": None,
            "Symbol": from_token_symbol
        },
        "ToToken": {
            "AmountChange": to_token_change,
            "AmountSOL": None,
            "AmountUSD": None,
            "Mint": to_token_mint,
            "PostSwapBalance": to_token_post_swap_bal,
            "PostSwapBalanceSOL": None,
            "PostSwapBalanceUSD": None,
            "PreSwapBalance": to_token_pre_swap_bal,
            "PreSwapBalanceSOL": None,
            "PreSwapBalanceUSD": None,
            "Symbol": to_token_symbol,
            "TokenSupply": None
        },
        "Transaction": data["Transaction"],
        "Block": data["Block"],
        "solUsdRate": data["solUsdRate"]
    }

    # 13) Enrich the data if needed
    swap_data = enricher(data, swap_data)

    return swap_data
