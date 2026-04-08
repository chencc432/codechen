# 基本数据结构
## 动态数组vector
### 初始化方法
```
#include <vector> 
int n = 7, m = 8;
// 初始化一个 int 型的空数组 
nums vector<int> nums; 
// 初始化一个大小为 n 的数组 nums，数组中的值默认都为 0
vector<int> nums(n); 
// 初始化一个元素为 1, 3, 5 的数组
nums vector<int> nums{1, 3, 5}; 
// 初始化一个大小为 n 的数组 nums，其值全都为 2 vector<int> nums(n, 2); 
// 初始化一个二维 int 数组 dp 
vector<vector<int>> dp; 
// 初始化一个大小为 m * n 的布尔数组 dp，
// 其中的值都初始化为 true 
vector<vector<bool>> dp(m, vector<bool>(n, true));
```
### 常用操作
```
判空，nums.empty();
大小，nums.size();
插入，nums.push_back();
最后一个元素引用：nums.back();
删除最后一个元素(无返回):nums.pop_back();
在索引 3 处插入一个元素 99 nums.insert(nums.begin() + 3, 99);
删除索引 2 处的元素 nums.erase(nums.begin() + 2);


```
## 队列queue
### 操作
```
定义：queue<int> q;
入队q.push();
判空q.empty();
队列大小：q.size()
队头队尾：q.front(),q.back();
删除对头：q.pop()
```
## 栈
```
stack<int>s;
s.push();
s.empty();
s.size();
s.top()
s.pop()

```
## 哈希表
```
unordered_map<int, string> hashmap;
unordered_map<int, string> hashmap{{1, "one"}, {2, "two"}, {3, "three"}};
在 C++ 的哈希表中，如果你访问一个不存在的键，它会自动创建这个键，对应的值是默认构造的值。
查找键hashmap.contains(2)
新增 haspmao[4]="four;
删除键hashmap.eraser(3);
遍历for (const auto &pair: hashmap) { cout << pair.first << " -> " << pair.second << endl; }
```
## 哈希集合
```
unordered_set<int> uset; // 初始化一个包含一些元素的哈希集合 set unordered_set<int> uset{1, 2, 3, 4};
存在：hashset.contains(3)；
// 插入一个新的元素 hashset.insert(5); // 删除一个元素 hashset.erase(2);
遍历for (const auto &element : hashset) { cout << element << endl; }
```
