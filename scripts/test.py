# scripts/swap.py

"""
swap.py

Processes classified Solana 'TRANSFER' or 'SWAP' transactions by extracting relevant details,
enriching them with token supply information from BitQuery, and preparing the data
for further handling.
"""

import json
import sys
import asyncio
import os

from tx_interpreter.functions.sol_swaps.script import sol_swaps
from tx_interpreter.functions.transfers.script import main
from tx_interpreter.functions.router.script import classify_transaction_detailed



def load_json_data(filename="dump/json9.json"):
    """
    Loads transaction data from a JSON file.

    Args:
        filename (str): Path to the JSON file containing transaction data.

    Returns:
        dict: Parsed JSON data as a Python dictionary.
    """
    try:
        with open(filename, "r", encoding="utf-8") as f:
            data = json.load(f)
            return data
    except Exception as e:
        return {}


async def async_main():
    """
    Asynchronous main function to process the transaction data.
    """
    try:
        input_data = sys.stdin.read()
        data = json.loads(input_data) # am I doing this part right?

        if not data:
            raise ValueError("No data loaded from JSON file.")
        
        txtype, mints = classify_transaction_detailed(data)

        if txtype == "SWAP":
            result = sol_swaps(data)
        elif txtype == "TRANSFER":
            if not mints:
                raise ValueError("No mints found for TRANSFER transaction.")
            result = await main(data, mints[0])
        else:
            result = {"message": f"Transaction type '{txtype}' is not supported."}
            
        # Serialize and print the result
        print(json.dumps(result))
    except Exception as e:
        error_message = {"error": str(e)}
        print(json.dumps(error_message))
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(async_main())
