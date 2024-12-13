# Error Wrapper Generator

A tool for generating error wrappers in Go that provides detailed error context including function name, arguments, and call chain.

## Features

- Automatically generates error wrapper types for functions that return errors
- Wraps errors with context about where and why they occurred
- Preserves error chains with `Unwrap()` support
- Includes function arguments in error messages
- Handles methods on structs and package-level functions
- Supports all basic Go types and custom types
- Automatically modifies existing error returns to use wrappers

## Installation

```bash
go install github.com/Bionic2113/errgen@latest
```

## Usage

Run in your project directory:

```bash
errgen
```

This will:
1. Scan all .go files in the current directory and subdirectories
2. Generate error wrapper types for functions that return errors
3. Create `errors.go` files in each package containing the wrappers
4. Modify existing error returns to use the new wrappers

## Example

Given this code:

```go
func ProcessUser(user *User, count int) error {
    if err := user.UpdateName("New"); err != nil {
        return err
    }
    return errors.New("processing failed")
}
```

The generator will create a wrapper and modify the code to:

```go
func ProcessUser(user *User, count int) error {
    if err := user.UpdateName("New"); err != nil {
        return NewProcessUserError(user, count, "user.UpdateName", err)
    }
    return NewProcessUserError(user, count, "processing failed", nil)
}

type ProcessUserError struct {
    user   *User
    count  int
    reason string
    err    error
}

func (e *ProcessUserError) Error() string {
    return "[pkg.ProcessUser] - ProcessUser - " + e.reason + 
           " - args: {user: " + fmt.Sprintf("%#v", e.user) + 
           ", count: " + strconv.Itoa(e.count) + "}\n" + 
           e.err.Error()
}

func (e *ProcessUserError) Unwrap() error {
    return e.err
}
```

This provides rich error context while maintaining the original error chain.

---

# Генератор оберток для ошибок

Инструмент для генерации оберток ошибок в Go, который предоставляет детальный контекст ошибки, включая имя функции, аргументы и цепочку вызовов.

## Возможности

- Автоматически генерирует типы оберток ошибок для функций, возвращающих ошибки
- Оборачивает ошибки с контекстом о том, где и почему они произошли
- Сохраняет цепочки ошибок с поддержкой `Unwrap()`
- Включает аргументы функции в сообщения об ошибках
- Обрабатывает методы структур и функции уровня пакета
- Поддерживает все базовые типы Go и пользовательские типы
- Автоматически модифицирует существующие возвраты ошибок для использования оберток

## Установка

```bash
go install github.com/Bionic2113/errgen@latest
```

## Использование

Запустите в директории вашего проекта:

```bash
errgen
```

Это:
1. Просканирует все .go файлы в текущей директории и поддиректориях
2. Сгенерирует типы оберток ошибок для функций, возвращающих ошибки
3. Создаст файлы `errors.go` в каждом пакете, содержащие обертки
4. Модифицирует существующие возвраты ошибок для использования новых оберток

## Пример

Для данного кода:

```go
func ProcessUser(user *User, count int) error {
    if err := user.UpdateName("New"); err != nil {
        return err
    }
    return errors.New("processing failed")
}
```

Генератор создаст обертку и модифицирует код в:

```go
func ProcessUser(user *User, count int) error {
    if err := user.UpdateName("New"); err != nil {
        return NewProcessUserError(user, count, "user.UpdateName", err)
    }
    return NewProcessUserError(user, count, "processing failed", nil)
}

type ProcessUserError struct {
    user   *User
    count  int
    reason string
    err    error
}

func (e *ProcessUserError) Error() string {
    return "[pkg.ProcessUser] - ProcessUser - " + e.reason + 
           " - args: {user: " + fmt.Sprintf("%#v", e.user) + 
           ", count: " + strconv.Itoa(e.count) + "}\n" + 
           e.err.Error()
}

func (e *ProcessUserError) Unwrap() error {
    return e.err
}
```

Это обеспечивает богатый контекст ошибки при сохранении оригинальной цепочки ошибок.
