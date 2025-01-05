import json
import sys

def detect_swap(balance_updates, signer_address):
    # Filter updates for the provided signer address and detect swaps
    filtered_updates = [
        update for update in balance_updates
        if (
            update["BalanceUpdate"]["Account"]["Address"] == signer_address or
            (
                update["BalanceUpdate"]["Account"].get("Token") and
                update["BalanceUpdate"]["Account"]["Token"]["Owner"] == signer_address
            )
        ) and (
            update["BalanceUpdate"]["PostBalance"] != update["BalanceUpdate"]["PreBalance"]
        )
    ]

    if len(filtered_updates) < 2:
        return {"swapDetected": False, "details": None}

    # Token1 is always SOL
    sol_update = next(
        (update for update in filtered_updates if update["Currency"].get("Symbol") == "SOL"), None
    )
    token_update = next(
        (update for update in filtered_updates if update["Currency"].get("Symbol") != "SOL"), None
    )

    if not sol_update or not token_update:
        return {"swapDetected": False, "details": None}

    # Extract values from BalanceUpdate for Token1 and Token2
    first_amount_raw = first_update["BalanceUpdate"]["PostBalance"] - first_update["BalanceUpdate"]["PreBalance"]
    last_amount_raw = last_update["BalanceUpdate"]["PostBalance"] - last_update["BalanceUpdate"]["PreBalance"]

    first_decimals = first_update["Currency"].get("Decimals", 0)
    last_decimals = last_update["Currency"].get("Decimals", 0)

    first_amount = first_amount_raw / (10 ** first_decimals)
    last_amount = last_amount_raw / (10 ** last_decimals)

    token1_mint = first_update["Currency"].get("MintAddress")
    token1_symbol = first_update["Currency"].get("Symbol", "null")
    token2_mint = last_update["Currency"].get("MintAddress")
    token2_symbol = last_update["Currency"].get("Symbol", "null")

    # Extract post-swap balance for Token2
    token2_post_swap_balance = last_update["BalanceUpdate"]["PostBalance"] / (10 ** last_decimals)

    # Extract transaction details
    transaction_fee = data.get("Transaction", {}).get("Fee", 0) / (10 ** 9)  # Convert lamports to SOL
    timestamp = data.get("Block", {}).get("Timestamp", 0)
    signature = data.get("Transaction", {}).get("Signature", "")

    # Check if SOL is involved and add SOL-specific fields
    def add_sol_fields(symbol, amount, decimals):
        return {
            "AmountSOL": amount if symbol == "SOL" else None,
            "PostSwapBalanceSOL": token1_post_swap_balance if symbol == "SOL" else None
        }

    # Build the resulting JSON structure
    result = {
        "TransactionDetails": {
            "Signer": signer_address,
            "Signature": signature,
            "Timestamp": timestamp
        },
        "Token1": {
            "Symbol": token1_symbol,
            "Mint": token1_mint,
            "Amount": abs(first_amount),
            "AmountUSD": None  # Placeholder for USD conversion
        },
        "Token2": {
            "Symbol": token2_symbol,
            "Mint": token2_mint,
            "Amount": abs(last_amount),
            "AmountUSD": None,  # Placeholder for USD conversion
            "PostSwapBalance": token2_post_swap_balance,
            "PostSwapBalanceUSD": None  # Placeholder for USD conversion
        },
        "Fee": {
            "Amount": transaction_fee,
            "AmountUSD": None  # Placeholder for USD conversion
        }
    }

    return {"swapDetected": True, "details": result}


if __name__ == "__main__":
    input_data = sys.stdin.read()
    try:
        data = json.loads(input_data)
        balance_updates = data.get("BalanceUpdates", [])
        signer = data.get("Transaction", {}).get("Signer")
        if not signer:
            raise ValueError("Signer address is required")

        result = detect_swap(balance_updates, signer)
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
