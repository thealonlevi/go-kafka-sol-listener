# transfers/bitquery.py

from typing import Any, Dict, Tuple

# BitQuery API configuration
BITQUERY_URL = "https://streaming.bitquery.io/eap"
BITQUERY_TOKEN = "ory_at_-mbOudJvKgJ1bSQ9upzINUpt1FHMbXHYd1Sa5yKs_ZU.DiWP8GQ7ZgbhrE1-Xdv0ibf32OfWJefg15cU0Y-mqMY"

import aiohttp
import asyncio
import json
import logging
import sys
# **Switch Event Loop Policy on Windows to Prevent RuntimeError**
if sys.platform.startswith('win'):
    asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())

async def fetch_dex_trade(token_mint: str) -> Dict[str, Any]:
    """
    Asynchronously fetches the latest DEX trade details for a specific token mint from BitQuery.

    Args:
        token_mint (str): The mint address of the token.

    Returns:
        Dict[str, Any]: The JSON response containing DEX trade details.

    Raises:
        ValueError: If the response contains errors or is malformed.
        aiohttp.ClientError: For network-related errors.
    """
    query = """
    query MyQuery($tokenMint: String!) {
      Solana {
        DEXTradeByTokens(
          limit: {count: 1}
          where: {Trade: {Currency: {MintAddress: {is: $tokenMint}}}}
          orderBy: {descending: Block_Slot}
        ) {
          Trade {
            Amount
            AmountInUSD
            Price
            PriceInUSD
          }
          Block {
            Date
            Time
          }
        }
      }
    }
    """

    variables = {
        "tokenMint": token_mint
    }

    payload = {
        "query": query,
        "variables": variables
    }

    headers = {
        'Content-Type': 'application/json',
        'Authorization': f'Bearer {BITQUERY_TOKEN}'
    }

    try:
        async with aiohttp.ClientSession() as session:
            logging.debug("Sending POST request to BitQuery.")
            async with session.post(BITQUERY_URL, json=payload, headers=headers) as response:
                logging.debug(f"Received response with status code {response.status}.")
                if response.status != 200:
                    text = await response.text()
                    logging.error(f"BitQuery request failed with status {response.status}: {response.reason}\nResponse: {text}")
                    raise ValueError(f"BitQuery request failed with status {response.status}: {response.reason}\nResponse: {text}")

                data = await response.json()

                if 'errors' in data:
                    logging.error(f"BitQuery returned errors: {data['errors']}")
                    raise ValueError(f"BitQuery returned errors: {data['errors']}")

                logging.debug("BitQuery request successful.")
                return data

    except aiohttp.ClientError as e:
        logging.error(f"Network-related error occurred: {e}")
        raise
    except asyncio.CancelledError:
        logging.warning("The fetch_dex_trade task was cancelled.")
        raise
    except Exception as e:
        logging.error(f"An unexpected error occurred: {e}")
        raise

async def fetch_token_supply(mint_address: str) -> Tuple[str, float]:
    """
    Asynchronously fetches the token's name and supply via BitQuery.

    Args:
        mint_address (str): The mint address of the token.
        bitquery_token (str): The BitQuery API token.

    Returns:
        Tuple[str, float]: A tuple containing the token name and post balance supply.

    Raises:
        ValueError: If the BitQuery token is not provided or the request fails.
    """
    bitquery_token = BITQUERY_TOKEN
    if not bitquery_token:
        logging.error("BitQuery token is not provided.")
        raise ValueError("BitQuery token is not provided.")

    url = "https://streaming.bitquery.io/eap"

    query = f"""
    {{
      Solana {{
        TokenSupplyUpdates(
          where: {{
            TokenSupplyUpdate: {{
              Currency: {{
                MintAddress: {{
                  is: "{mint_address}"
                }}
              }}
            }}
          }}
          limit: {{count: 1}}
          orderBy: {{}}
        ) {{
          TokenSupplyUpdate {{
            Amount
            Currency {{
              MintAddress
              Name
              Symbol
            }}
            PostBalance
            PostBalanceInUSD
            PreBalanceInUSD
            PreBalance
          }}
        }}
      }}
    }}
    """

    payload = {
        "query": query,
        "variables": {}
    }

    headers = {
        'Content-Type': 'application/json',
        'Authorization': f"Bearer {bitquery_token}"
    }

    try:
        async with aiohttp.ClientSession() as session:
            logging.debug("Sending POST request to BitQuery.")
            async with session.post(url, json=payload, headers=headers) as response:
                logging.debug(f"Received response with status code {response.status}.")
                if response.status != 200:
                    text = await response.text()
                    logging.error(f"BitQuery request failed with status {response.status}: {response.reason}\nResponse: {text}")
                    raise ValueError(f"BitQuery request failed with status {response.status}: {response.reason}\nResponse: {text}")

                data = await response.json()

                # Parse response to extract name and supply
                solana = data.get("data", {}).get("Solana")
                if not solana:
                    logging.warning("No 'Solana' data found in BitQuery response.")
                    return "", 0.0

                token_supply_updates = solana.get("TokenSupplyUpdates", [])
                if not token_supply_updates:
                    logging.warning(f"No 'TokenSupplyUpdates' found for mint address {mint_address}.")
                    return "", 0.0

                ts_update = token_supply_updates[0].get("TokenSupplyUpdate")
                if not ts_update:
                    logging.warning("No 'TokenSupplyUpdate' found in the response.")
                    return "", 0.0

                currency = ts_update.get("Currency")
                if not currency:
                    logging.warning("No 'Currency' data found in 'TokenSupplyUpdate'.")
                    return "", 0.0

                name = currency.get("Symbol")
                if not name:
                    logging.warning("No 'Name' found in 'Currency'.")
                    return "", 0.0

                post_balance = ts_update.get("PostBalance")
                if post_balance is None:
                    logging.warning("No 'PostBalance' found in 'TokenSupplyUpdate'.")
                    return "", 0.0

                return name, float(post_balance)

    except aiohttp.ClientError as e:
        logging.error(f"Network-related error occurred: {e}")
        raise
    except asyncio.CancelledError:
        logging.warning("The fetch_token_supply task was cancelled.")
        raise
    except Exception as e:
        logging.error(f"An unexpected error occurred: {e}")
        raise

def fetch_token_price(token_mint):
        dex_trade_data = fetch_dex_trade(token_mint)
        dex_trade = dex_trade_data['data']['Solana']['DEXTradeByTokens'][0]['Trade']
        tokenusd = float(dex_trade.get('PriceInUSD', 0))
        return tokenusd
    
# Optional: Example usage when running the script directly
# This allows the script to be used both as an importable module and as a standalone script.

async def _main():
    token_mint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"  # Example token mint
    try:
        dex_trade_data = await fetch_dex_trade(token_mint)
        dex_trade = dex_trade_data['data']['Solana']['DEXTradeByTokens'][0]['Trade']
        tokenusd = float(dex_trade.get('PriceInUSD', 0))

        # Unpack the tuple correctly
        token_name, tokensupply = await fetch_token_supply(token_mint, BITQUERY_TOKEN)
        print(f"Token Name: {token_name}")
        print(f"Token Supply: {tokensupply}")
        print(f"Token Price(USD): {tokenusd}")

        # Calculate Market Capitalization
        tokenmc = tokenusd * tokensupply
        print(f"Market Capitalization for {token_name}: {tokenmc}")

    except Exception as e:
        logging.error(f"Error fetching DEX trade data: {e}")

if __name__ == "__main__":
    try:
        asyncio.run(_main())
    except RuntimeError as e:
        logging.error(f"RuntimeError: {e}")