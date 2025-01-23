from .base_filter import BaseFilter

class TokenFilter(BaseFilter):
    def __init__(self, data, tokens):
        super().__init__(data)
        self.tokens = tokens

    def apply(self):
        """
        Filters the data to include only updates with specific tokens.
        """
        
        data = self.data['BalanceUpdates']
        return [
            update for update in data
            if (
                update.get("Currency").get("MintAddress") and
                update["Currency"]["MintAddress"] in self.tokens
            )
        ]
