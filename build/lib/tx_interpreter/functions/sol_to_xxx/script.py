import copy
from tx_interpreter.filters import BalanceChangeFilter, TokenFilter, SignerFilter
from tx_interpreter.utils import BalanceCalculator, DominantFigureFilter, BalanceUpdateLocator, get_currencies_involved
from tx_interpreter.functions.enricher.script import enricher

# Define the SOL mint address
SOL_MINT = "11111111111111111111111111111111"

def sol_to_xxx(data):
    """
    Processes a transaction to detect SOL-to-token or token-to-SOL swaps and enriches the data with additional details.

    Args:
        data (dict): The transaction data containing balance updates, transaction details, etc.

    Returns:
        dict: A dictionary containing enriched swap details.
    """
    
    # Create deep copies of the input data for various filters
    signer_data = copy.deepcopy(data)
    first_token_data = copy.deepcopy(data)
    sol_token_data = copy.deepcopy(data)

    # Filter balance updates to only include those related to the signer
    signer_data['BalanceUpdates'] = SignerFilter(signer_data).apply()

    # Further copies for SOL and first token processing
    sol_token_signer_data = copy.deepcopy(signer_data)
    first_token_signer_data = copy.deepcopy(signer_data)

    # Identify the tokens involved in the transaction
    tokens = get_currencies_involved(signer_data)

    # Ensure exactly two tokens are involved, one of which must be SOL
    if len(tokens) > 2:
        raise ValueError("[ERROR] Use xxx_to_xxx function for multi-token swaps.")

    is_sol_present = SOL_MINT in tokens
    if not is_sol_present:
        raise ValueError("[ERROR] Use xxx_to_xxx function for non-SOL swaps.")

    # Determine the indices for SOL and the other token
    sol_index = tokens.index(SOL_MINT)
    token_index = 1 - sol_index

    # Apply token filters to isolate balance updates for each token
    first_token_data['BalanceUpdates'] = TokenFilter(data, tokens[token_index]).apply()
    sol_token_data['BalanceUpdates'] = TokenFilter(data, tokens[sol_index]).apply()
    sol_token_signer_data['BalanceUpdates'] = TokenFilter(signer_data, tokens[sol_index]).apply()
    first_token_signer_data['BalanceUpdates'] = TokenFilter(signer_data, tokens[token_index]).apply()

    # Calculate balance differences for SOL and the other token
    first_token_differences = BalanceCalculator.calculate_balance_differences(first_token_data)
    sol_token_differences = BalanceCalculator.calculate_balance_differences(sol_token_data)
    sol_token_differences_signer = BalanceCalculator.calculate_balance_differences(sol_token_signer_data)
    first_token_differences_signer = BalanceCalculator.calculate_balance_differences(first_token_signer_data)

    # Determine if a swap was detected
    data['swapDetected'] = (
        (sum(sol_token_differences_signer) < 0 and sum(first_token_differences_signer) > 0) or
        (sum(sol_token_differences_signer) > 0 and sum(first_token_differences_signer) < 0)
    )

    # Initialize variables for swap fee and direction
    swap_fees = 0
    sol_to_token = sum(sol_token_differences_signer) > 0

    # Identify dominant figures in balance changes
    first_dom_figures = DominantFigureFilter.filter_dominant_figures(first_token_differences)
    sol_dom_figures = DominantFigureFilter.filter_dominant_figures(sol_token_differences)

    # Ensure only one dominant figure is present for each token
    if len(first_dom_figures) > 1 or len(sol_dom_figures) > 1:
        raise ValueError("[ERROR] Failed to analyze swap.")

    # Process the swap based on its direction (SOL-to-token or token-to-SOL)
    if not sol_to_token:
        swap_fees = abs(min(sol_token_differences) / 1_000_000_000 + sol_dom_figures[0] / 1_000_000_000)
        from_token_change = sol_dom_figures[0] / 1_000_000_000
        to_token_change = first_dom_figures[0] / 1_000_000
        from_token_mint = SOL_MINT
        to_token_mint = tokens[token_index]
        from_token_symbol = "SOL"
        to_token_symbol = first_token_signer_data['BalanceUpdates'][0]['Currency']['Symbol']

        to_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data['BalanceUpdates'], max(first_token_differences_signer))
        to_token_pre_swap_bal = to_token_update['BalanceUpdate']['PreBalance'] / 1_000_000
        to_token_post_swap_bal = to_token_update['BalanceUpdate']['PostBalance'] / 1_000_000

        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data['BalanceUpdates'], min(sol_token_differences_signer))
        from_token_pre_swap_bal = sol_token_update['BalanceUpdate']['PreBalance'] / 1_000_000_000
        from_token_post_swap_bal = sol_token_update['BalanceUpdate']['PostBalance'] / 1_000_000_000

    else:
        swap_fees = abs(min(sol_token_differences) / 1_000_000_000)
        from_token_change = first_dom_figures[0] / 1_000_000
        to_token_change = sol_dom_figures[0] / 1_000_000_000
        from_token_mint = tokens[token_index]
        to_token_mint = SOL_MINT
        to_token_symbol = "SOL"
        from_token_symbol = first_token_signer_data['BalanceUpdates'][0]['Currency']['Symbol']

        from_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            first_token_signer_data['BalanceUpdates'], min(first_token_differences_signer))
        from_token_pre_swap_bal = from_token_update['BalanceUpdate']['PreBalance'] / 1_000_000
        from_token_post_swap_bal = from_token_update['BalanceUpdate']['PostBalance'] / 1_000_000

        sol_token_update = BalanceUpdateLocator().find_balance_update_by_amount(
            sol_token_signer_data['BalanceUpdates'], max(sol_token_differences_signer))
        to_token_pre_swap_bal = sol_token_update['BalanceUpdate']['PreBalance'] / 1_000_000_000
        to_token_post_swap_bal = sol_token_update['BalanceUpdate']['PostBalance'] / 1_000_000_000

    # Construct the swap data
    swap_data = {
        "BlockchainFees": {
            "Amount": data['Transaction']['Fee'],
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
        "Transaction": data['Transaction'],
        "Block": data['Block'],
        "solUsdRate": data["solUsdRate"]
    }

    # Enrich the swap data further
    swap_data = enricher(data, swap_data)

    return swap_data