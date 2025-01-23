import json

class BalanceCalculator:
    @staticmethod
    def calculate_balance_differences(data):
        """
        Calculate differences between PostBalance and PreBalance for each BalanceUpdate.

        Args:
            data (dict): JSON data containing filtered balance updates.

        Returns:
            list: A list of differences rounded to 7 decimal places.
        """
        try:
            # Initialize a list to store differences
            balance_differences = []

            # Loop through each BalanceUpdate and calculate PostBalance - PreBalance
            for balance_update in data.get("BalanceUpdates", []):
                balance = balance_update.get("BalanceUpdate", {})
                post_balance = balance.get("PostBalance", 0)
                pre_balance = balance.get("PreBalance", 0)

                # Calculate the difference and round to 7 decimal places
                difference = round(post_balance - pre_balance, 7)
                balance_differences.append(difference)

            return balance_differences

        except KeyError as e:
            print(f"Missing key in data: {e}")
            return []

    @staticmethod
    def display_balance_differences(balance_differences):
        """
        Display the balance differences in a formatted way.

        Args:
            balance_differences (list): A list of balance differences.
        """
        print("Balance Differences:")
        for diff in balance_differences:
            print(f"{diff:.7f}")
