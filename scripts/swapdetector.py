import json
import sys

def detect_swap(balance_updates, signer_address):
    # Your detection logic here
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

    first_update = filtered_updates[0]
    last_update = filtered_updates[-1]

    first_amount_raw = first_update["BalanceUpdate"]["PostBalance"] - first_update["BalanceUpdate"]["PreBalance"]
    last_amount_raw = last_update["BalanceUpdate"]["PostBalance"] - last_update["BalanceUpdate"]["PreBalance"]

    first_decimals = first_update["Currency"].get("Decimals", 0)
    last_decimals = last_update["Currency"].get("Decimals", 0)

    first_amount = first_amount_raw / (10 ** first_decimals)
    last_amount = last_amount_raw / (10 ** last_decimals)

    token1_mint = first_update["Currency"].get("MintAddress")
    token1_name = first_update["Currency"].get("Name", "null")
    token2_mint = last_update["Currency"].get("MintAddress")
    token2_name = last_update["Currency"].get("Name", "null")

    if first_amount < 0:
        spent = f"-{abs(first_amount):.5f} {token1_name or token1_mint}"
        received = f"+{abs(last_amount):.5f} {token2_name or token2_mint}"
    else:
        spent = f"-{abs(last_amount):.5f} {token2_name or token2_mint}"
        received = f"+{abs(first_amount):.5f} {token1_name or token1_mint}"

    return {"swapDetected": True, "details": f"Swapped: {received} {spent}"}


if __name__ == "__main__":
    input_data = sys.stdin.read()
    try:
        data = json.loads(input_data)
        balance_updates = data.get("BalanceUpdates", [])
        signer = data.get("Transaction", {}).get("Signer")
        if not signer:
            raise ValueError("Signer address is required")

        result = detect_swap(balance_updates, signer)
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
