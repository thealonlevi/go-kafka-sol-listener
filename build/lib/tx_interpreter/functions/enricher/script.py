import requests
import json

def generate_one_line_query(mint_address: str) -> str:
    """
    Generate a one-line GraphQL query string for a given mint address.

    Args:
        mint_address (str): The mint address to query.

    Returns:
        str: Escaped one-line GraphQL query.
    """
    multi_line_str = f"""
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

    escaped_query = (
        multi_line_str
        .strip()
        .replace('"', '\\"')
        .replace('\n', '\\n')
    )
    return escaped_query

def fetch_token_supply(mint_address: str, bitquery_token: str) -> (str, str):
    """
    Fetch the token's name and supply via BitQuery, mirroring the Go code logic.
    Returns a tuple of (token_name, post_balance_supply).

    Raises an exception if any required data is missing or if the request fails.
    """

    if not bitquery_token:
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
    payload = json.dumps({
        "query": query,
        "variables": "{}"
    })

    headers = {
        'Content-Type': 'application/json',
        'Authorization': f"Bearer {bitquery_token}"
    }

    # Send the request
    response = requests.request("POST", url, data=payload, headers=headers)
    if response.status_code != 200:
        raise ValueError(f"BitQuery request failed with status {response.status_code}")
    
    data = response.json()
    
    # Parse response to extract name and supply
    solana = data.get("data", {}).get("Solana")
    err = False
    if not solana:
        err = True
        

    token_supply_updates = solana.get("TokenSupplyUpdates", [])
    if not token_supply_updates:
        err = True
        return "", 0
    try:
        ts_update = token_supply_updates[0].get("TokenSupplyUpdate")
    except:
        err = True
        return "", 0
    
    if not ts_update:
        err = True

    currency = ts_update.get("Currency")
    if not currency:
        err = True

    name = currency.get("Name")
    if not name:
        err = True
    post_balance = ts_update.get("PostBalance")

    if post_balance is None:
        err = True
    
    if err:
        return "", 0
    else:
        return name, post_balance


def enrich_token_supply(tx_details: dict, bitquery_token: str) -> dict:
    """
    Enrich 'ToToken' within tx_details by fetching its token supply from BitQuery.
    Updates 'Symbol' and 'TokenSupply' in place if a valid mint is found and is not SOL.

    Returns the modified tx_details.
    """
    tosol = False
    if tx_details['ToToken']['Mint'] == "11111111111111111111111111111111": tosol = True

    
    if tosol:
        to_token = tx_details['FromToken']
    else:
        to_token = tx_details['ToToken']
        
    mint = to_token['Mint']

    # Skip if there's no mint address or it's the SOL mint address
    if not mint or mint == "11111111111111111111111111111111":
        return tx_details
      
    name, supply = fetch_token_supply(mint, bitquery_token)
    if not name == "":
        to_token["Symbol"] = name
    to_token["TokenSupply"] = float(supply)

    
    if tosol:
        tx_details['FromToken'] = to_token
    else:
        tx_details['ToToken'] = to_token
    
    return tx_details

def enricher(base_data: dict, swap_data: dict) -> dict:
    """
    Enrich swap data with SOL-to-USD calculations and additional token details.

    Args:
        base_data (dict): Base transaction data.
        swap_data (dict): Swap transaction data.

    Returns:
        dict: Enriched swap data.
    """
    # Enrich swap data with token supply
    swap_data = enrich_token_supply(
        swap_data,
        "ory_at_-mbOudJvKgJ1bSQ9upzINUpt1FHMbXHYd1Sa5yKs_ZU.DiWP8GQ7ZgbhrE1-Xdv0ibf32OfWJefg15cU0Y-mqMY"
    )

    # Set SOL to USD conversion rate
    sol_usd_ratio = base_data['solUsdRate']

    # Determine if swap is to SOL
    is_to_sol = swap_data['ToToken']['Mint'] == "11111111111111111111111111111111"

    # Enrich based on the type of swap (to SOL or from SOL)
    if is_to_sol:
        sol_amount = swap_data['ToToken']['AmountChange']
        token_amount = swap_data['FromToken']['AmountChange']
        sol_token_ratio = token_amount / sol_amount

        # Enrich ToToken (SOL)
        swap_data['ToToken']['AmountSOL'] = sol_amount
        swap_data['ToToken']['AmountUSD'] = sol_amount * sol_usd_ratio

        swap_data['ToToken']['PostSwapBalanceSOL'] = swap_data['ToToken']['PostSwapBalance']
        swap_data['ToToken']['PostSwapBalanceUSD'] = swap_data['ToToken']['PostSwapBalance'] * sol_usd_ratio

        swap_data['ToToken']['PreSwapBalanceSOL'] = swap_data['ToToken']['PreSwapBalance']
        swap_data['ToToken']['PreSwapBalanceUSD'] = swap_data['ToToken']['PreSwapBalance'] * sol_usd_ratio

        # Enrich FromToken (Token)
        swap_data['FromToken']['AmountSOL'] = token_amount / sol_token_ratio
        swap_data['FromToken']['AmountUSD'] = (token_amount / sol_token_ratio) * sol_usd_ratio

        swap_data['FromToken']['PostSwapBalanceSOL'] = swap_data['FromToken']['PostSwapBalance'] / sol_token_ratio
        swap_data['FromToken']['PostSwapBalanceUSD'] = (swap_data['FromToken']['PostSwapBalance'] / sol_token_ratio) * sol_usd_ratio

        swap_data['FromToken']['PreSwapBalanceSOL'] = swap_data['FromToken']['PreSwapBalance'] / sol_token_ratio
        swap_data['FromToken']['PreSwapBalanceUSD'] = (swap_data['FromToken']['PreSwapBalance'] / sol_token_ratio) * sol_usd_ratio

    else:
        token_amount = swap_data['ToToken']['AmountChange']
        sol_amount = swap_data['FromToken']['AmountChange']
        sol_token_ratio = token_amount / sol_amount

        # Enrich FromToken (SOL)
        swap_data['FromToken']['AmountSOL'] = sol_amount
        swap_data['FromToken']['AmountUSD'] = sol_amount * sol_usd_ratio

        swap_data['FromToken']['PostSwapBalanceSOL'] = swap_data['FromToken']['PostSwapBalance']
        swap_data['FromToken']['PostSwapBalanceUSD'] = swap_data['FromToken']['PostSwapBalance'] * sol_usd_ratio

        swap_data['FromToken']['PreSwapBalanceSOL'] = swap_data['FromToken']['PreSwapBalance']
        swap_data['FromToken']['PreSwapBalanceUSD'] = swap_data['FromToken']['PreSwapBalance'] * sol_usd_ratio

        # Enrich ToToken (Token)
        swap_data['ToToken']['AmountSOL'] = token_amount / sol_token_ratio
        swap_data['ToToken']['AmountUSD'] = (token_amount / sol_token_ratio) * sol_usd_ratio

        swap_data['ToToken']['PostSwapBalanceSOL'] = swap_data['ToToken']['PostSwapBalance'] / sol_token_ratio
        swap_data['ToToken']['PostSwapBalanceUSD'] = (swap_data['ToToken']['PostSwapBalance'] / sol_token_ratio) * sol_usd_ratio

        swap_data['ToToken']['PreSwapBalanceSOL'] = swap_data['ToToken']['PreSwapBalance'] / sol_token_ratio
        swap_data['ToToken']['PreSwapBalanceUSD'] = (swap_data['ToToken']['PreSwapBalance'] / sol_token_ratio) * sol_usd_ratio

    # Enrich fees
    swap_data['BlockchainFees']['AmountUSD'] = swap_data['BlockchainFees']['Amount'] * sol_usd_ratio
    swap_data['SwapFees']['AmountUSD'] = swap_data['SwapFees']['Amount'] * sol_usd_ratio

    # Detect swap occurrence
    from_token_change = swap_data['FromToken'].get('AmountChange', 0)
    to_token_change = swap_data['ToToken'].get('AmountChange', 0)
    swap_data['swapDetected'] = from_token_change != 0 and to_token_change != 0

    return swap_data
