import json
import sys
from tx_interpreter.functions.sol_to_xxx.script import sol_to_xxx

if __name__ == "__main__":
    try:
        input_data = sys.stdin.read()
        data = json.loads(input_data)
        result = sol_to_xxx(data)
        print(json.dumps(result))  # Ensure the output is JSON serialized
    except Exception as e:
        error_message = {"error": str(e)}
        print(json.dumps(error_message))
        sys.exit(1)
