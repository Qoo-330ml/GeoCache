import base64


def encrypt(text: str) -> str:
    """加密字符串：反转 + Base64编码"""
    reversed_text = text[::-1]
    encrypted = base64.b64encode(reversed_text.encode()).decode()
    return encrypted


def main():
    print("=== GeoCache 配置加密工具 ===")
    print("加密算法：字符串反转 + Base64编码\n")
    
    while True:
        print("\n请选择操作：")
        print("1. 加密 URL")
        print("2. 加密 API 密钥")
        print("3. 加密自定义字符串")
        print("4. 批量加密")
        print("0. 退出")
        
        choice = input("\n请输入选项 (0-4): ").strip()
        
        if choice == "0":
            print("再见！")
            break
        
        elif choice == "1":
            url = input("请输入 URL (例如: https://geocache.pdzhou.top): ").strip()
            if url:
                encrypted = encrypt(url)
                print(f"\n原始 URL: {url}")
                print(f"加密结果: {encrypted}")
            else:
                print("URL 不能为空")
        
        elif choice == "2":
            api_key = input("请输入 API 密钥: ").strip()
            if api_key:
                encrypted = encrypt(api_key)
                print(f"\n原始密钥: {api_key}")
                print(f"加密结果: {encrypted}")
            else:
                print("API 密钥不能为空")
        
        elif choice == "3":
            text = input("请输入要加密的字符串: ").strip()
            if text:
                encrypted = encrypt(text)
                print(f"\n原始字符串: {text}")
                print(f"加密结果: {encrypted}")
            else:
                print("字符串不能为空")
        
        elif choice == "4":
            print("\n批量加密模式（输入空行结束）：")
            items = []
            while True:
                item = input(f"请输入第 {len(items) + 1} 个字符串（或直接回车结束）: ").strip()
                if not item:
                    break
                items.append(item)
            
            if items:
                print("\n批量加密结果：")
                for i, item in enumerate(items, 1):
                    encrypted = encrypt(item)
                    print(f"{i}. 原始: {item}")
                    print(f"   加密: {encrypted}")
                    print()
            else:
                print("没有输入任何字符串")
        
        else:
            print("无效的选项，请重新选择")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n程序已中断")
    except Exception as e:
        print(f"\n发生错误: {e}")
