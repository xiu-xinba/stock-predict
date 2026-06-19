# 商业标准代码注释 Spec

## Why
项目代码目前缺乏系统性的注释，大部分文件没有包级说明、函数文档、类型说明或字段注释，不符合商业软件的可维护性标准。为所有项目代码添加符合商业标准的注释，可显著提升代码可读性、团队协作效率和后续维护体验。

## What Changes
- Go 后端：为所有包添加 `// Package xxx ...` 包级注释，为所有导出类型/函数/方法添加 godoc 标准注释，为关键未导出函数添加注释，为结构体字段添加说明注释
- Vue 前端：为所有 TypeScript 类型/接口添加 JSDoc 注释，为所有导出函数/composable 添加 JSDoc 注释，为 Vue 组件添加 `<script>` 块顶部组件说明注释，为复杂逻辑添加行内注释
- Python AKShare 微服务：为所有模块添加 docstring，为所有函数添加 Google 风格 docstring
- 不修改任何代码逻辑、不改变代码行为，仅添加注释

## Impact
- Affected code: 后端全部 Go 源文件、前端全部 TypeScript/Vue 源文件、AKShare Python 源文件
- 不影响编译、构建、测试或运行时行为

## ADDED Requirements

### Requirement: Go 包级注释
每个 Go 包目录下的主文件 SHALL 包含 `// Package xxx ...` 格式的包注释，说明该包的职责和核心功能。

#### Scenario: 包注释存在
- **WHEN** 检查任意 Go 包目录
- **THEN** 该包至少一个文件包含以 `// Package <packagename>` 开头的包注释，描述包的用途

### Requirement: Go 导出符号注释
所有导出的类型、函数、方法、常量、变量 SHALL 添加符合 godoc 标准的注释，以符号名开头，说明其用途、参数含义和返回值。

#### Scenario: 导出函数注释
- **WHEN** 检查任意导出函数
- **THEN** 该函数上方有以 `// FuncName ...` 开头的注释，说明其功能

#### Scenario: 导出类型注释
- **WHEN** 检查任意导出类型（struct/interface/type alias）
- **THEN** 该类型上方有以 `// TypeName ...` 开头的注释，说明其用途

### Requirement: Go 结构体字段注释
结构体中含义不明显的导出字段 SHALL 添加行尾注释说明其含义和单位。

#### Scenario: 字段注释
- **WHEN** 检查结构体中的导出字段
- **THEN** 含义不直观的字段（如缩写、业务术语）有行尾 `// ...` 注释说明

### Requirement: Go 关键未导出函数注释
关键未导出函数（如核心算法、复杂业务逻辑、辅助函数）SHALL 添加注释说明其用途。

#### Scenario: 未导出函数注释
- **WHEN** 检查超过 20 行的未导出函数
- **THEN** 该函数上方有注释说明其功能和关键逻辑

### Requirement: TypeScript 类型/接口 JSDoc 注释
所有导出的 TypeScript interface、type、enum SHALL 添加 JSDoc 注释，说明其用途和字段含义。

#### Scenario: 接口注释
- **WHEN** 检查任意导出的 TypeScript interface
- **THEN** 该接口上方有 `/** ... */` JSDoc 注释，说明其用途，关键字段有 `@remarks` 或行内说明

### Requirement: TypeScript 导出函数 JSDoc 注释
所有导出的函数、composable、store 定义 SHALL 添加 JSDoc 注释，包含 `@param`、`@returns` 等标签。

#### Scenario: 函数注释
- **WHEN** 检查任意导出的 TypeScript 函数
- **THEN** 该函数上方有 JSDoc 注释，包含功能说明和参数描述

### Requirement: Vue 组件说明注释
每个 Vue 组件的 `<script setup>` 或 `<script>` 块顶部 SHALL 添加组件用途说明注释。

#### Scenario: 组件注释
- **WHEN** 检查任意 `.vue` 文件
- **THEN** `<script>` 区域顶部有注释说明该组件的功能和用途

### Requirement: Python 模块和函数 Docstring
AKShare 微服务的所有 Python 模块 SHALL 添加模块级 docstring，所有函数 SHALL 添加 Google 风格 docstring（含 Args/Returns/Raises）。

#### Scenario: Python 函数 docstring
- **WHEN** 检查任意 Python 函数
- **THEN** 该函数有 Google 风格 docstring，包含功能说明和参数/返回值描述

### Requirement: 注释语言统一
所有注释 SHALL 使用中文编写，技术术语可保留英文原文。

#### Scenario: 注释语言
- **WHEN** 检查任意新增注释
- **THEN** 注释内容使用中文，技术术语（如 API 名称、协议术语）可保留英文

### Requirement: 不改变代码逻辑
添加注释过程 SHALL NOT 修改任何代码逻辑、变量名、函数签名或运行时行为。

#### Scenario: 代码逻辑不变
- **WHEN** 注释添加完成后运行 `go vet`、`go build`、前端 `typecheck` 和 `build`
- **THEN** 所有命令均通过，无新增错误或警告
