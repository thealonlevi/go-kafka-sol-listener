import copy
from tx_interpreter.filters import (
    TokenFilter,
    SignerFilter,
    NotSignerFilter
)
from tx_interpreter.utils import (
    BalanceCalculator,
    DominantFigureFilter,
    BalanceUpdateLocator,
    get_currencies_involved,
    fee_calculator  # We import fee_calculator but do NOT redefine it
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


def sol_swaps(data):
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
    signer_data, notsigner_data = copy.deepcopy(data), copy.deepcopy(data)
    first_token_data = copy.deepcopy(data)
    sol_token_data = copy.deepcopy(data)

    # 2) Filter out only the signer's balance updates
    signer_data["BalanceUpdates"] = SignerFilter(signer_data).apply()
    notsigner_data["BalanceUpdates"] = NotSignerFilter(notsigner_data).apply()

    # 3) Additional copies to isolate SOL vs. other token from the signer
    sol_token_signer_data = copy.deepcopy(signer_data)
    first_token_signer_data = copy.deepcopy(signer_data)
    
    sol_token_notsigner_data = copy.deepcopy(notsigner_data)
    first_token_notsigner_data = copy.deepcopy(notsigner_data)

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
    sol_token_data["BalanceUpdates"] = TokenFilter(data, tokens[sol_index]).apply()

    # ...and for the signer-specific data
    first_token_signer_data["BalanceUpdates"] = TokenFilter(signer_data, tokens[token_index]).apply()
    sol_token_signer_data["BalanceUpdates"] = TokenFilter(signer_data, tokens[sol_index]).apply()
    
    first_token_notsigner_data["BalanceUpdates"] = TokenFilter(notsigner_data, tokens[token_index]).apply()
    sol_token_notsigner_data["BalanceUpdates"] = TokenFilter(notsigner_data, tokens[sol_index]).apply()

    # 6) Compute decimal-based differences for each token
    first_token_differences = BalanceCalculator.calculate_balance_differences(first_token_data)
    sol_token_differences   = BalanceCalculator.calculate_balance_differences(sol_token_data)
    first_token_diffs_signer = BalanceCalculator.calculate_balance_differences(first_token_signer_data)
    sol_token_diffs_signer   = BalanceCalculator.calculate_balance_differences(sol_token_signer_data)
    first_token_diffs_notsigner = BalanceCalculator.calculate_balance_differences(first_token_notsigner_data)
    sol_token_diffs_notsigner   = BalanceCalculator.calculate_balance_differences(sol_token_notsigner_data)

    # 7) Detect a swap
    data["swapDetected"] = (
        (sum(sol_token_diffs_signer) < 0 and sum(first_token_diffs_signer) > 0)
        or (sum(sol_token_diffs_signer) > 0 and sum(first_token_diffs_signer) < 0)
    )

    # 8) Decide direction: If signer ends with more SOL => token->SOL; otherwise => SOL->token
    sol_to_token = (sum(sol_token_diffs_signer) > 0)

    # 9) Find the dominant figure for each token
    first_dom_figs = DominantFigureFilter.filter_dominant_figures(first_token_diffs_notsigner)
    sol_dom_figs   = DominantFigureFilter.filter_dominant_figures(sol_token_diffs_notsigner)

    # Expect only one dominant figure per token
    if len(first_dom_figs) > 1 or len(sol_dom_figs) > 1:
        raise ValueError("[ERROR] Failed to analyze swap.")

    # 10) Retrieve decimals
    first_token_decimals = get_token_decimals(first_token_signer_data["BalanceUpdates"], default=9)
    sol_decimals = 9  # Known for SOL

    # 11) Convert the dominant figures from raw amounts
    # (ex: negative if parted with that token)
    if not sol_to_token:
        # SOL -> Token
        from_token_symbol = "SOL"
        from_token_mint   = SOL_MINT
        to_token_symbol   = first_token_signer_data["BalanceUpdates"][0]["Currency"]["Symbol"]
        to_token_mint     = tokens[token_index]

        from_token_change = parse_amount(sol_dom_figs[0], sol_decimals)
        to_token_change   = parse_amount(first_dom_figs[0], first_token_decimals)

        # Identify relevant pre/post updates
        to_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data["BalanceUpdates"],
            max(first_token_diffs_signer)
        )
        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data["BalanceUpdates"],
            min(sol_token_diffs_signer)
        )

        from_token_pre_swap_bal  = parse_amount(sol_token_update["BalanceUpdate"]["PreBalance"],  sol_decimals)
        from_token_post_swap_bal = parse_amount(sol_token_update["BalanceUpdate"]["PostBalance"], sol_decimals)
        to_token_pre_swap_bal    = parse_amount(to_token_update["BalanceUpdate"]["PreBalance"], first_token_decimals)
        to_token_post_swap_bal   = parse_amount(to_token_update["BalanceUpdate"]["PostBalance"], first_token_decimals)

    else:
        # Token -> SOL
        from_token_symbol = first_token_signer_data["BalanceUpdates"][0]["Currency"]["Symbol"]
        from_token_mint   = tokens[token_index]
        to_token_symbol   = "SOL"
        to_token_mint     = SOL_MINT

        from_token_change = parse_amount(first_dom_figs[0], first_token_decimals)
        to_token_change   = parse_amount(sol_dom_figs[0],   sol_decimals)

        from_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data["BalanceUpdates"],
            min(first_token_diffs_signer)
        )
        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data["BalanceUpdates"],
            max(sol_token_diffs_signer)
        )

        from_token_pre_swap_bal  = parse_amount(from_token_update["BalanceUpdate"]["PreBalance"],  first_token_decimals)
        from_token_post_swap_bal = parse_amount(from_token_update["BalanceUpdate"]["PostBalance"], first_token_decimals)
        to_token_pre_swap_bal    = parse_amount(sol_token_update["BalanceUpdate"]["PreBalance"],   sol_decimals)
        to_token_post_swap_bal   = parse_amount(sol_token_update["BalanceUpdate"]["PostBalance"],  sol_decimals)

    # 12) Use fee_calculator to get chain & trade fees (replacing local fee logic)
    chain_fee, trade_fee = fee_calculator(data)
    chain_fee = parse_amount(chain_fee, sol_decimals)
    trade_fee = parse_amount(trade_fee, sol_decimals)

    # 13) Build final payload
    swap_data = {
        "BlockchainFees": {
            "Amount": chain_fee,
            "AmountUSD": None
        },
        "SwapFees": {
            "Amount": trade_fee,
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

    # 14) Enrich and return
    return enricher(data, swap_data)
