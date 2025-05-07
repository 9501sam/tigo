import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

# Load CSV
df = pd.read_csv("service_calls.csv")

# Pivot the data to matrix form
pivot = df.pivot(index="from", columns="to", values="count").fillna(0)

# Plot heatmap
plt.figure(figsize=(10, 8))
sns.heatmap(pivot, annot=True, fmt=".0f", cmap="Reds")
plt.title("Service-to-Service Call Heatmap")
plt.ylabel("Caller (From)")
plt.xlabel("Callee (To)")
plt.tight_layout()

plt.savefig("heatmap.png", dpi=300)  # Change filename and dpi as needed

plt.show()
