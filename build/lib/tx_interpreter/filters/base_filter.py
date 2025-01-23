class BaseFilter:
    def __init__(self, data):
        self.data = data

    def apply(self):
        """
        Apply the filter logic.
        Must be implemented by subclasses.
        """
        raise NotImplementedError("Subclasses must implement the 'apply' method.")
