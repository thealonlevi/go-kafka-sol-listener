class BalanceUpdateLocator:
    @staticmethod
    def find_balance_update_by_amount(balance_updates, target_amount):
        """
        Finds the BalanceUpdate with the exact PostBalance - PreBalance amount or closest to it.

        Args:
            balance_updates (list): List of BalanceUpdate objects.
            target_amount (float): The amount to search for.

        Returns:
            dict: The BalanceUpdate object closest to the target amount.
        """
        if not balance_updates:
            return None

        closest_update = None
        closest_difference = float('inf')  # Initialize with a very large number

        for update in balance_updates:
            balance = update.get("BalanceUpdate", {})
            post_balance = balance.get("PostBalance", 0)
            pre_balance = balance.get("PreBalance", 0)
            difference = post_balance - pre_balance

            # Calculate the absolute difference between the target and this amount
            abs_difference = abs(difference - target_amount)

            # Update the closest match if this is smaller
            if abs_difference < closest_difference:
                closest_difference = abs_difference
                closest_update = update

        return closest_update
