import json
import sys
from tx_interpreter.functions.sol_swaps.script import sol_swaps
from tx_interpreter.functions.transfers.script import main
from tx_interpreter.functions.router.script import classify_transaction_detailed

if __name__ == "__main__":
    try:
        input_data = sys.stdin.read()
        data = json.loads(input_data)
        txtype, mints = classify_transaction_detailed(input_data)
        if txtype == "SWAP":
            result = sol_swaps(data)
        elif txtype == "TRANSFER":
            result = main(data, mints[0])

        print(json.dumps(result))  # Ensure the output is JSON serialized
    except Exception as e:
        error_message = {"error": str(e)}
        print(json.dumps(error_message))
        sys.exit(1)
