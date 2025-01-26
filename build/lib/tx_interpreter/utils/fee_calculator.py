import json
import copy

# Filters
from tx_interpreter.filters import (
    TokenFilter,
    SignerFilter,
    NotSignerFilter,
    BalanceChangeFilter
)

# Utilities
from tx_interpreter.utils import (
    BalanceCalculator,
    DominantFigureFilter,
    get_currencies_involved
)


def load_data_from_file(filename="dump/json.json"):
    """
    Loads transaction data from a JSON file.
    Assumes the file contains valid JSON matching the required structure.
    """
    with open(filename, "r", encoding="utf-8") as f:
        data = json.load(f)
    return data


def prepare_token_data(data):
    """
    Partitions the transaction data into signer and non-signer subsets,
    removes trivial balance changes, then separates each subset by token.

    Returns:
        (token1_signerdata, token2_signerdata,
         token1_notsignerdata, token2_notsignerdata,
         signer_data, notsigner_data, tokens)
    """
    # 1) Create copies
    signer_data = copy.deepcopy(data)
    notsigner_data = copy.deepcopy(data)

    # 2) Filter updates: signer vs. non-signer
    signer_data["BalanceUpdates"] = SignerFilter(signer_data).apply()
    notsigner_data["BalanceUpdates"] = NotSignerFilter(notsigner_data).apply()

    # 3) Remove zero/trivial changes
    signer_data["BalanceUpdates"] = BalanceChangeFilter(signer_data).apply()
    notsigner_data["BalanceUpdates"] = BalanceChangeFilter(notsigner_data).apply()

    # 4) Identify tokens
    tokens = get_currencies_involved(signer_data)
    if len(tokens) < 2:
        raise ValueError("Expected at least two tokens in signer data.")

    # 5) Prepare structures for each token (signer vs. non-signer)
    token1_signerdata = copy.deepcopy(signer_data)
    token2_signerdata = copy.deepcopy(signer_data)
    token1_notsignerdata = copy.deepcopy(notsigner_data)
    token2_notsignerdata = copy.deepcopy(notsigner_data)

    # 6) Filter each copy by token
    token1_signerdata["BalanceUpdates"] = TokenFilter(signer_data, tokens[0]).apply()
    token2_signerdata["BalanceUpdates"] = TokenFilter(signer_data, tokens[1]).apply()
    token1_notsignerdata["BalanceUpdates"] = TokenFilter(notsigner_data, tokens[0]).apply()
    token2_notsignerdata["BalanceUpdates"] = TokenFilter(notsigner_data, tokens[1]).apply()

    return (
        token1_notsignerdata,
        tokens
    )


def fee_calculator(data):
    """
    Calculates chain and trade fees from transaction data.

    Args:
        data (dict): Transaction data, including 'BalanceUpdates'.
        swap_amount (float): The SOL swap amount (user input or output).

    Returns:
        (ChainFee, TradeFee) as floats.
    """
    # Extract subsets
    (
        token1_notsignerdata,
        tokens
    ) = prepare_token_data(data)

    # Ensure SOL is tokens[0] if present
    if tokens[0] != "11111111111111111111111111111111":
        (
            token1_notsignerdata,
            tokens
        ) = prepare_token_data(data)

    # Non-signer differences
    token1_outputbalances = BalanceCalculator().calculate_balance_differences(token1_notsignerdata)
    
    print(token1_outputbalances)

    # Identify dominant figures (liquidity pools, etc.)
    token1_liquiditypool = DominantFigureFilter().filter_dominant_figures(token1_outputbalances)
    print("Token1 Liquidity Pool: ", token1_liquiditypool)

    # Example fee calculation approach
    if abs(sum(token1_liquiditypool))>abs(sum(token1_outputbalances)):
        TradeFee = abs(sum(token1_liquiditypool))-abs(sum(token1_outputbalances))
    else:
        TradeFee = abs(sum(token1_outputbalances)) - abs(sum(token1_liquiditypool))
    
    ChainFee = data["Transaction"].get("Fee", 0)

    return ChainFee, TradeFee
