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

    # Extract values from BalanceUpdate
    sol_amount_raw = sol_update["BalanceUpdate"]["PostBalance"] - sol_update["BalanceUpdate"]["PreBalance"]
    token_amount_raw = token_update["BalanceUpdate"]["PostBalance"] - token_update["BalanceUpdate"]["PreBalance"]

    sol_decimals = sol_update["Currency"].get("Decimals", 0)
    token_decimals = token_update["Currency"].get("Decimals", 0)

    sol_amount = sol_amount_raw / (10 ** sol_decimals)
    token_amount = token_amount_raw / (10 ** token_decimals)

    sol_pre_balance = sol_update["BalanceUpdate"]["PreBalance"] / (10 ** sol_decimals)
    sol_post_balance = sol_update["BalanceUpdate"]["PostBalance"] / (10 ** sol_decimals)

    token_pre_balance = token_update["BalanceUpdate"]["PreBalance"] / (10 ** token_decimals)
    token_post_balance = token_update["BalanceUpdate"]["PostBalance"] / (10 ** token_decimals)

    sol_mint = sol_update["Currency"].get("MintAddress")
    sol_symbol = sol_update["Currency"].get("Symbol", "null")
    token_mint = token_update["Currency"].get("MintAddress")
    token_symbol = token_update["Currency"].get("Symbol", "null")

    # Extract transaction details
    transaction_fee = data.get("Transaction", {}).get("Fee", 0) / (10 ** 9)  # Convert from lamports to SOL
    timestamp = data.get("Block", {}).get("Timestamp", 0)
    signature = data.get("Transaction", {}).get("Signature", "")

    # Build the resulting JSON structure
    result = {
        "TransactionDetails": {
            "Signer": signer_address,
            "Signature": signature,
            "Timestamp": timestamp
        },
        "Token1": {
            "Symbol": sol_symbol,
            "Mint": sol_mint,
            "AmountChange": sol_amount,  # Positive or negative
            "PreSwapBalance": sol_pre_balance,
            "PreSwapBalanceUSD": None,  # Placeholder for USD conversion
            "PreSwapBalanceSOL": sol_pre_balance,
            "PostSwapBalance": sol_post_balance,
            "PostSwapBalanceUSD": None,  # Placeholder for USD conversion
            "PostSwapBalanceSOL": sol_post_balance
        },
        "Token2": {
            "Symbol": token_symbol,
            "Mint": token_mint,
            "AmountChange": token_amount,  # Positive or negative
            "PreSwapBalance": token_pre_balance,
            "PreSwapBalanceUSD": None,  # Placeholder for USD conversion
            "PreSwapBalanceSOL": None,  # Placeholder for SOL conversion if applicable
            "PostSwapBalance": token_post_balance,
            "PostSwapBalanceUSD": None,  # Placeholder for USD conversion
            "PostSwapBalanceSOL": None  # Placeholder for SOL conversion if applicable
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
