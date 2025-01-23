from .base_filter import BaseFilter

class BalanceChangeFilter(BaseFilter):
    def apply(self):
        """
        Filters the data to include only updates where PostBalance != PreBalance.
        """
        data = self.data['BalanceUpdates']
        return [
            update for update in data
            if (
                isinstance(update, dict) and  # Ensure each item is a dictionary
                "BalanceUpdate" in update and  # Ensure the key "BalanceUpdate" exists
                "PostBalance" in update["BalanceUpdate"] and
                "PreBalance" in update["BalanceUpdate"] and
                update["BalanceUpdate"]["PostBalance"] != update["BalanceUpdate"]["PreBalance"]
            )
        ]
