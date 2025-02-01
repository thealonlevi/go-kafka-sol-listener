# transfers/script.py

"""
script.py

Processes classified Solana 'TRANSFER' transactions by extracting relevant details,
enriching them with token supply information from BitQuery, and preparing the data
for further handling.
"""

import asyncio
import json
import sys
from typing import Any, Dict, Tuple, Optional

import argparse
import os

from tx_interpreter.functions.transfers.bitquery import fetch_dex_trade, fetch_token_supply

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
    fee_calculator,
    get_token_decimals,
    parse_amount
)
from tx_interpreter.functions.enricher.script import enricher

# Define native mints used for fees (e.g., SOL and WSOL)
NATIVE_MINTS = {
    "11111111111111111111111111111111",             # SOL
    "So11111111111111111111111111111111111111112"  # WSOL
}

# Define sets of equivalent mints (e.g., SOL and WSOL)
EQUIVALENT_MINTS = [
    {"11111111111111111111111111111111", "So11111111111111111111111111111111111111112"},  # SOL and WSOL
    # Add more sets if there are other equivalent mints
]

# Define auxiliary program addresses to exclude (e.g., Tokenkeg, ComputeBudget, SysvarRent)
AUXILIARY_PROGRAM_ADDRESSES = {
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
    "ComputeBudget111111111111111111111111111111",
    "SysvarRent111111111111111111111111111111111",
    # Add more auxiliary program addresses if needed
}


def get_equivalent_mint(mint: str) -> str:
    """
    Determines the representative mint for a given mint by checking equivalent sets.

    Args:
        mint (str): The mint address to check.

    Returns:
        str: The representative mint address.
    """
    for equivalent_set in EQUIVALENT_MINTS:
        if mint in equivalent_set:
            # Always return SOL as the representative if present
            if "11111111111111111111111111111111" in equivalent_set:
                return "11111111111111111111111111111111"
            else:
                representative = next(iter(equivalent_set))
                return representative
    return mint  # Return the original mint if no equivalence found


def fetch_sol_usd_rate() -> float:
    """
    Fetches the current SOL to USD conversion rate from CoinGecko.

    Returns:
        float: The SOL to USD rate.

    Raises:
        ValueError: If the API request fails or the response is malformed.
        aiohttp.ClientError: For network-related errors.
    """
    import aiohttp

    url = "https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd"
    try:
        async def get_rate():
            async with aiohttp.ClientSession() as session:
                async with session.get(url) as response:
                    if response.status != 200:
                        text = await response.text()
                        raise ValueError(f"CoinGecko request failed with status {response.status}: {response.reason}\nResponse: {text}")

                    data = await response.json()
                    sol_usd_rate = data.get("solana", {}).get("usd")
                    if sol_usd_rate is None:
                        raise ValueError("SOL to USD rate not found in CoinGecko response.")
                    return float(sol_usd_rate)

        return asyncio.run(get_rate())
    except Exception as e:
        return 0  # Fallback rate


def token_to_usd(price_usd: float, amount: float) -> float:
    """
    Converts token amount to USD using the provided price.

    Args:
        price_usd (float): The price of one token in USD.
        amount (float): The amount of tokens.

    Returns:
        float: The equivalent amount in USD.
    """
    usd = abs(amount) * abs(price_usd)
    return usd


def process_transfer(data: Dict[str, Any], mint: str, token_price_usd: float, sol_usd_rate: float) -> Optional[Dict[str, Any]]:
    """
    Processes a classified 'TRANSFER' transaction to extract relevant details.

    Args:
        data (dict): The raw transaction data.
        mint (str): The mint address of the transferred token.
        token_price_usd (float): The price of the token in USD.
        sol_usd_rate (float): The SOL to USD conversion rate.

    Returns:
        dict or None: The structured transfer data if successful, else None.
    """
    transfer_details = {
        "MintAddress": "",
        "Symbol": "",
        "Decimals": 0,
        "Sender": {
            "PublicKey": "",
            "PreTransferBalance": 0,
            "PostTransferBalance": 0
        },
        "Receiver": {
            "PublicKey": "",
            "PreTransferBalance": 0,
            "PostTransferBalance": 0
        },
        "AmountTransferred": 0.0,
        "AmountTransferredSOL": 0.0,
        "AmountTransferredUSD": 0.0
    }

    # Apply TokenFilter to get relevant balance updates for the specified mint
    filtered_balance_updates = TokenFilter(data, [mint]).apply()

    # Calculate balance differences
    updates = BalanceCalculator.calculate_balance_differences(data)

    if not updates:
        return None

    outgoing = min(updates)  # How much of the token was sent
    ingoing = max(updates)   # How much of the token was received

    
    # Locate the corresponding balance updates
    outgoing_balanceupdate = BalanceUpdateLocator().find_balance_update_by_amount(filtered_balance_updates, outgoing)
    ingoing_balanceupdate = BalanceUpdateLocator().find_balance_update_by_amount(filtered_balance_updates, ingoing)

    if not outgoing_balanceupdate or not ingoing_balanceupdate:
        return None

    # Get token decimals
    decimals = get_token_decimals([outgoing_balanceupdate])

    # Identify sender
    sender = data.get('Transaction', {}).get('Signer', "")
    # Identify receiver
    token_info = ingoing_balanceupdate.get('BalanceUpdate', {}).get('Account', {}).get('Token')
    if token_info:
        receiver = token_info.get('Owner', ingoing_balanceupdate['BalanceUpdate']['Account']['Address'])
    else:
        receiver = ingoing_balanceupdate['BalanceUpdate']['Account']['Address']
    
    # Parse the amounts
    amount_transferred = parse_amount(outgoing, decimals)  # Actual parsed amount of tokens transferred
    amount_transferred_usd = token_to_usd(token_price_usd, amount_transferred)  # USD value
    amount_transferred_sol = amount_transferred_usd / sol_usd_rate  # Equivalent in SOL

    # Populate transfer details
    transfer_details["MintAddress"] = mint
    transfer_details["Symbol"] = ingoing_balanceupdate['Currency'].get('Symbol', '')
    transfer_details["Decimals"] = decimals
    transfer_details["Sender"] = {
        "PublicKey": sender,
        "PreTransferBalance": outgoing_balanceupdate['BalanceUpdate'].get('PreBalance', 0),
        "PostTransferBalance": outgoing_balanceupdate['BalanceUpdate'].get('PostBalance', 0)
    }
    transfer_details["Receiver"] = {
        "PublicKey": receiver,
        "PreTransferBalance": ingoing_balanceupdate['BalanceUpdate'].get('PreBalance', 0),
        "PostTransferBalance": ingoing_balanceupdate['BalanceUpdate'].get('PostBalance', 0)
    }
    transfer_details["AmountTransferred"] = abs(round(amount_transferred, decimals))
    transfer_details["AmountTransferredSOL"] = abs(round(amount_transferred_sol, decimals))
    transfer_details["AmountTransferredUSD"] = abs(round(amount_transferred_usd, 6))

    structured_transfer = {
        "TransferDetails": transfer_details,
        "swapDetected": False  # As this is a transfer
    }

    return structured_transfer


async def enrich_with_token_supply(structured_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Enriches the structured data with token supply information by fetching it from BitQuery.

    Args:
        structured_data (dict): The structured transaction data.

    Returns:
        dict: The enriched transaction data.
    """
    try:
        # Extract the mint address from TransferDetails
        mint_address = structured_data["TransferDetails"]["MintAddress"]
        # Skip native mints as they don't require enrichment
        if mint_address in NATIVE_MINTS:
            return structured_data

        # Fetch token supply information
        name, supply = await fetch_token_supply(mint_address)
        
        # Update structured data
        structured_data["TransferDetails"]["Symbol"] = name
        structured_data["TransferDetails"]["TokenSupply"] = supply

        return structured_data

    except ValueError as ve:
        return structured_data
    except Exception as e:
        return structured_data


async def main(raw_data, transferred_token_mint):
    """
    Main function to read raw transfer data from a JSON file, process it,
    enrich it with token supply information, fetch dynamic SOL to USD rate,
    and print the structured data.
    """


    # Fetch dynamic SOL to USD rate
    sol_usd_rate = float(raw_data['solUsdRate'])

    # Fetch DEX Trade Data to get token price
    try:
        dex_trade_data = await fetch_dex_trade(transferred_token_mint)
        # Extract PriceInUSD from DEX trade data
        dex_trade = dex_trade_data['data']['Solana']['DEXTradeByTokens'][0]['Trade']
        tokenusd = float(dex_trade.get('PriceInUSD', 0))
    except Exception as e:
        tokenusd = 0.0  # Fallback or handle accordingly

    # Process the transfer transaction
    structured_transfer = process_transfer(raw_data, transferred_token_mint, tokenusd, sol_usd_rate)

    if not structured_transfer:
        sys.exit(0)

    # Enrich the structured data with token supply information
    enriched_data = await enrich_with_token_supply(structured_transfer)

    # Include Block and Transaction details from the raw data
    enriched_data["Block"] = raw_data.get("Block", {})
    enriched_data["Transaction"] = raw_data.get("Transaction", {})
    enriched_data["solUsdRate"] = sol_usd_rate
    
    # Calculate Market Capitalization
    tokenusd_final = enriched_data['TransferDetails'].get('AmountTransferredUSD', 0.0)
    tokensupply = enriched_data['TransferDetails'].get('TokenSupply', 0.0)
    tokenmc = tokenusd_final * tokensupply
    enriched_data["MarketCapitalization"] = tokenmc
    
    return enriched_data

