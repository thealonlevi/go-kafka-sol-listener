from .base_filter import BaseFilter

class NotSignerFilter(BaseFilter):
    def __init__(self, data):
        """
        Initialize the NotSignerFilter with the data.
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
        Filters out any updates that involve the signer address (or a token whose
        Owner is the signer). In other words, it returns only updates where neither
        the account address nor the token owner match the signer address.
        """
        updates = self.data.get("BalanceUpdates", [])
        return [
            update for update in updates
            if (
                isinstance(update, dict)
                and "BalanceUpdate" in update
                and "Account" in update["BalanceUpdate"]
                # Must satisfy BOTH conditions to be considered "not signer"
                and update["BalanceUpdate"]["Account"]["Address"] != self.signer_address
                and (
                    not update["BalanceUpdate"]["Account"].get("Token")
                    or update["BalanceUpdate"]["Account"]["Token"].get("Owner") != self.signer_address
                )
            )
        ]
