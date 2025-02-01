def get_token_decimals(balance_updates, default=9):
    """
    Safely extract the `Decimals` field from a list of balance updates.
    Returns `default` if not found or invalid.
    """
    if not balance_updates or len(balance_updates) == 0:
        return default

    currency_info = balance_updates[0].get("Currency", {})
    decimals = currency_info.get("Decimals", default)
    try:
        decimals = int(decimals)
    except (TypeError, ValueError):
        decimals = default

    # Bound the decimals to avoid extreme or negative values
    if decimals < 0 or decimals > 18:
        decimals = default

    return decimals


def parse_amount(raw_amount, decimals):
    """
    Convert a raw integer-like amount to a float by dividing by 10^decimals.
    """
    return float(raw_amount) / (10 ** decimals)