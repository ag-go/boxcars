//go:build (!js || !wasm) && !android

package game

const DefaultServerAddress = "tcp://bgammon.org:1337"
const OptimizeDraws = true
