import numpy as np

class DominantFigureFilter:
    @staticmethod
    def filter_dominant_figures(balance_differences, threshold_ratio=0.1):
        """
        Filters out figures that are too small relative to the most dominant figure,
        considering both positive and negative values by magnitude.

        Args:
            balance_differences (list of float): List of balance differences 
                                                 (can be positive or negative).
            threshold_ratio (float): Ratio threshold for determining dominance.
                                     Defaults to 0.1 (10%).

        Returns:
            list: A list of dominant figures (both positive and negative) 
                  whose absolute value is at least threshold_ratio * max|difference|.
        """
        if not balance_differences:
            return []

        # Convert to NumPy array for easier manipulation
        differences = np.array(balance_differences, dtype=float)

        # Find the largest magnitude among both positive and negative values
        max_abs_value = np.max(np.abs(differences))

        # Filter values whose magnitude is above the threshold ratio
        dominant_figures = [
            float(value)
            for value in differences
            if abs(value) >= max_abs_value * threshold_ratio
        ]

        return dominant_figures

    @staticmethod
    def display_dominant_figures(dominant_figures):
        """
        Displays the dominant figures in a formatted list, including negative values.
        
        Args:
            dominant_figures (list of float): A list of dominant figures.
        """
        print("Dominant Figures:")
        for figure in dominant_figures:
            print(f"{figure:.7f}")
