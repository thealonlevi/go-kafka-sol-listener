import numpy as np

class DominantFigureFilter:
    @staticmethod
    def filter_dominant_figures(balance_differences, threshold_ratio=0.1):
        """
        Filters out the minuscule amounts and retains only the dominant figures.

        Args:
            balance_differences (list): List of balance differences (positive values only).
            threshold_ratio (float): Ratio threshold for determining dominance. Defaults to 0.1 (10%).

        Returns:
            list: A list of dominant figures.
        """
        if not balance_differences:
            return []

        # Convert to NumPy array for easier manipulation
        differences = np.array(balance_differences)

        # Calculate the max value in the list
        max_value = np.max(differences)

        # Filter values that are at least threshold_ratio of the max value
        dominant_figures = [float(value) for value in differences if float(value) >= max_value * threshold_ratio]

        return dominant_figures

    @staticmethod
    def display_dominant_figures(dominant_figures):
        """
        Displays the dominant figures in a formatted list.

        Args:
            dominant_figures (list): A list of dominant figures.
        """
        print("Dominant Figures:")
        for figure in dominant_figures:
            print(f"{figure:.7f}")
