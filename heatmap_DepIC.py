import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

# 設定 matplotlib 顯示中文 (如果您需要標籤顯示中文的話)
# plt.rcParams['font.sans-serif'] = ['Noto Sans CJK TC'] # 例如 'Noto Sans CJK TC' 或 'SimHei'
# plt.rcParams['axes.unicode_minus'] = False # 解決負號顯示問題

# Load CSV
# 確保 'depICs.csv' 文件存在於您的 Python 腳本的相同目錄下
try:
    df = pd.read_csv("depICs.csv")
except FileNotFoundError:
    print("Error: depICs.csv not found. Please ensure the Go program generated it in the same directory.")
    exit()

# 檢查 DataFrame 內容
# print(df.head())
# print(df.info())

# Pivot the data to matrix form
# 確保 'values' 參數指向 CSV 中的正確列名，即 Go 程式碼中輸出的第三列名稱
pivot = df.pivot(index="from", columns="to", values="DepIC_Value").fillna(0)

# 如果您想確保所有服務都出現在軸上，即使它們沒有顯式出現在數據中（例如 DepIC 為 0 的情況）
# 您可以從 Go 程式碼中獲取完整的服務列表，並用它來 reindex pivot 表格
# 例如，如果 Go 中的 common.Services 列表是固定的：
all_services = ["cartservice", "checkoutservice", "currencyservice", "emailservice",
                "frontend", "paymentservice", "productcatalogservice", "recommendationservice",
                "redis-cart", "shippingservice"]
pivot = pivot.reindex(index=all_services, columns=all_services, fill_value=0)


# Plot heatmap
plt.figure(figsize=(12, 10)) # 稍微增大圖形尺寸以確保標籤清晰
sns.heatmap(pivot,
            annot=True,     # 在每個儲存格中顯示數值
            fmt=".2f",      # 將格式從 ".0f" 改為 ".2f"，顯示浮點數兩位小數
            cmap="viridis", # 顏色映射，"viridis" 通常比 "Reds" 效果更好，顏色漸變更豐富
            linewidths=.5,  # 增加網格線
            linecolor='black', # 網格線顏色
            cbar_kws={'label': 'DepIC Value'}) # 顏色條標籤

plt.title("Service-to-Service DepIC Heatmap", fontsize=16) # 調整標題字體大小
plt.ylabel("Base Service (From)", fontsize=12) # 調整 Y 軸標籤字體大小
plt.xlabel("Dependent Service (To)", fontsize=12) # 調整 X 軸標籤字體大小

plt.xticks(rotation=45, ha='right', fontsize=10) # 旋轉 x 軸標籤，使其更易讀，調整字體大小
plt.yticks(rotation=0, fontsize=10)   # 確保 y 軸標籤水平，調整字體大小

plt.tight_layout() # 自動調整佈局，防止標籤重疊

plt.savefig("heatmap_DepIC.png", dpi=300) # 更改文件名和 dpi
print("Heatmap saved to heatmap_DepIC.png")

plt.show() # 顯示圖表
