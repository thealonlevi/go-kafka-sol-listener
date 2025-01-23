def get_currencies_involved(data):
    """
    Extracts and returns a list of unique currencies involved in the swap.

    Args:
        data (dict): The JSON data containing BalanceUpdates.

    Returns:
        list: A list of unique mint addresses representing currencies involved in the swap.
    """
    try:
        # Initialize a set to store unique mint addresses
        unique_currencies = set()

        # Iterate over BalanceUpdates
        for update in data.get("BalanceUpdates", []):
            currency = update.get("Currency", {})
            mint_address = currency.get("MintAddress")

            if mint_address:
                unique_currencies.add(mint_address)

        # Convert the set to a sorted list for consistent output
        return sorted(unique_currencies)

    except Exception as e:
        print(f"Error while extracting currencies: {e}")
        return []
