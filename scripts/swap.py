import json
import sys
from tx_interpreter.functions.sol_swaps.script import sol_swaps

if __name__ == "__main__":
    try:
        input_data = sys.stdin.read()
        data = json.loads(input_data)
        result = sol_swaps(data)
        print(json.dumps(result))  # Ensure the output is JSON serialized
    except Exception as e:
        error_message = {"error": str(e)}
        print(json.dumps(error_message))
        sys.exit(1)
