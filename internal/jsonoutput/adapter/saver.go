package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// Saver は収集結果を JSON と改行にして writer へ出力する。
// NewSaver で組み立てる。共有 writer への同時書き込みの同期は呼び出し側が担当する。
type Saver struct{ writer io.Writer }

// NewSaver は利用可能な writer を受け取る。nil は拒否する。
// writer の Close は呼び出し側が担当し、Saver は所有権を引き継がない。
func NewSaver(writer io.Writer) (*Saver, error) {
	if writer == nil {
		return nil, errors.New("JSON output writer is required")
	}

	return &Saver{writer: writer}, nil
}

// SaveCollection は JSON を1回書き込み、原因を保持したエラーを返す。再試行しない。
// 書き込み前に context を確認するが、通常の writer の実行中の Write は中断できない。
// エラー時に一部が書かれた可能性があり、原子的な出力・永続化は保証しない。
func (s *Saver) SaveCollection(ctx context.Context, result steamdomain.Collection) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("prepare collection JSON output: %w", err)
	}

	data, err := MarshalCollection(result)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("write collection JSON: %w", err)
	}

	data = append(data, '\n')
	written, err := s.writer.Write(data)
	if err != nil {
		return fmt.Errorf("write collection JSON: %w", err)
	}

	if written != len(data) {
		return fmt.Errorf("write collection JSON: %w", io.ErrShortWrite)
	}

	return nil
}
