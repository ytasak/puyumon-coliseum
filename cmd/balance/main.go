// Command balance はroster combinationの総当たりsimulationを回し、集計結果をJSONで出力する。
//
// 集計は internal/balance が行う。ここではflagの読み取りと出力だけを行い、
// 指標の定義も判定もコマンド側へ持たない。
//
// 出力したJSONはそのままbalance分析の入力になる。同じflagからは必ず同じ結果が出る。
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/ytasak/puyumon-coliseum/internal/balance"
)

func main() {
	firstSeed := flag.Uint64("first-seed", 0, "first seed")
	trials := flag.Int("trials", 10, "battles per ordered matchup")
	maxTurns := flag.Int("max-turns", 0, "turn limit per battle (0 uses the default)")
	flag.Parse()

	report, err := balance.Run(balance.Config{
		FirstSeed: *firstSeed,
		Trials:    *trials,
		MaxTurns:  *maxTurns,
	})
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatalf("encode report: %v", err)
	}
}
