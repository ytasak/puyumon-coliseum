package game

import (
	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// characters は6体の見た目。SpeciesIDから引く。
//
// **実機確認のための仮の見た目で、デザインの確定ではない。** 座標と倍率は
// 目で見て合わせた暫定値で、YTA-31では「6体を見分けられること」と
// 「Composite Sprite層がそのまま使えること」だけを確かめる。
//
// 画像素材は足さない。体を1つのEmojiで作り、表情と装飾を重ねるという
// YTA-8で確かめたやり方をそのまま使う。
var characters = map[battle.SpeciesID]sprite.Character{
	roster.SpeciesBull: sprite.NewCharacter(
		sprite.Part{Emoji: "🐂", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "😤", X: 0.02, Y: -0.26, Scale: 0.34, Z: 1},
		sprite.Part{Emoji: "💢", X: 0.30, Y: -0.32, Scale: 0.26, Rotation: 0.2, Z: 2},
	),
	roster.SpeciesStar: sprite.NewCharacter(
		sprite.Part{Emoji: "⭐", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "🥺", X: 0, Y: -0.04, Scale: 0.36, Z: 1},
		sprite.Part{Emoji: "💧", X: -0.30, Y: 0.20, Scale: 0.24, Rotation: -0.2, Z: 2},
	),
	roster.SpeciesJolt: sprite.NewCharacter(
		sprite.Part{Emoji: "⚡", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "🤪", X: 0.04, Y: -0.10, Scale: 0.34, Z: 1},
		sprite.Part{Emoji: "💨", X: -0.32, Y: 0.14, Scale: 0.26, Rotation: -0.15, Z: 2},
	),
	roster.SpeciesPalm: sprite.NewCharacter(
		sprite.Part{Emoji: "🌴", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "😫", X: -0.04, Y: -0.24, Scale: 0.32, Z: 1},
		sprite.Part{Emoji: "💤", X: 0.30, Y: -0.30, Scale: 0.24, Rotation: 0.15, Z: 2},
	),
	roster.SpeciesWhale: sprite.NewCharacter(
		sprite.Part{Emoji: "🐋", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "🥺", X: -0.14, Y: -0.12, Scale: 0.30, Z: 1},
		sprite.Part{Emoji: "❄️", X: 0.30, Y: -0.26, Scale: 0.26, Z: 2},
	),
	roster.SpeciesCharm: sprite.NewCharacter(
		sprite.Part{Emoji: "💋", X: 0, Y: 0, Scale: 1, Z: 0},
		sprite.Part{Emoji: "😘", X: -0.02, Y: -0.26, Scale: 0.34, Z: 1},
		sprite.Part{Emoji: "💫", X: 0.30, Y: -0.30, Scale: 0.24, Rotation: 0.2, Z: 2},
	),
}

// characterFor はSpeciesIDの見た目を返す。
//
// 定義が無ければ代表Emoji 1文字だけの見た目を返す。キャラクターが増えても
// 描画側に分岐を足さずに済むようにしておく。
func characterFor(species battle.SpeciesID) sprite.Character {
	if character, ok := characters[species]; ok {
		return character
	}
	return sprite.NewCharacter(sprite.Part{Emoji: emojiFor(species), Scale: 1})
}

// emojiFor は一覧や控えの表示に使う代表Emojiを返す。
func emojiFor(species battle.SpeciesID) string {
	character, ok := roster.Lookup(species)
	if !ok {
		return ""
	}
	return character.Emoji
}
