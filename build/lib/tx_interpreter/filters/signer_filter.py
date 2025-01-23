from .base_filter import BaseFilter

class SignerFilter(BaseFilter):
    def __init__(self, data):
        """
        Initialize the SignerFilter with the data.
        Extracts the signer address from the 'Transaction' section.
        """
        super().__init__(data)
        self.signer_address = self._get_signer_address()

    def _get_signer_address(self):
        """
        Extracts the signer address from the data.
        Assumes the signer is located at 'Transaction -> Signer'.
        """
        transaction = self.data.get("Transaction", {})
        signer = transaction.get("Signer")
        if not signer:
            raise ValueError("No signer address found in the 'Transaction' section.")
        return signer

    def apply(self):
        """
        Filters the data to include only updates related to the signer address.
        """
        data = self.data['BalanceUpdates']  # Focus on 'BalanceUpdates' section
        return [
            update for update in data
            if (
                isinstance(update, dict) and  # Ensure each item is a dictionary
                "BalanceUpdate" in update and  # Ensure the key "BalanceUpdate" exists
                "Account" in update["BalanceUpdate"] and  # Ensure 'Account' exists
                (
                    update["BalanceUpdate"]["Account"]["Address"] == self.signer_address or
                    (
                        update["BalanceUpdate"]["Account"].get("Token") and
                        update["BalanceUpdate"]["Account"]["Token"].get("Owner") == self.signer_address
                    )
                )
            )
        ]
