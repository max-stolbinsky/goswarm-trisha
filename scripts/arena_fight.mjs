#!/usr/bin/env node
// Run GoSwarm test-arena fights via WebSocket. Usage:
//   node arena_fight.mjs --token TOKEN --mode deathmatch --opp starter --seed 138234
import fs from 'fs'
import { parseArgs } from 'util'

const { values: args } = parseArgs({
  options: {
    token: { type: 'string' },
    mode: { type: 'string', default: 'deathmatch' },
    opp: { type: 'string', default: 'starter' },
    seed: { type: 'string', default: '138234' },
    code: { type: 'string', default: '' },
    codeFile: { type: 'string', default: '' },
    speed: { type: 'string', default: '8' },
    timeoutMs: { type: 'string', default: '120000' },
  },
})

const token = args.token
if (!token) {
  console.error('missing --token')
  process.exit(2)
}
const playerCode = args.codeFile ? fs.readFileSync(args.codeFile, 'utf8') : args.code
if (!playerCode) {
  console.error('missing --code or --codeFile')
  process.exit(2)
}

const bots = JSON.parse(fs.readFileSync('/tmp/goswarm_bots.json', 'utf8'))
const opp = bots.find((b) => b.id === args.opp)
if (!opp) {
  console.error('unknown opp', args.opp)
  process.exit(2)
}

const seed = Number(args.seed)
const mode = args.mode
const speed = Number(args.speed)
const timeoutMs = Number(args.timeoutMs)

const wsUrl = 'wss://goswarm.dev/ws/arena'
const ws = new WebSocket(wsUrl, {
  headers: { Authorization: `Bearer ${token}` },
})

const logs = []
let done = null
let lastTick = 0
let playerError = null

const timer = setTimeout(() => {
  console.error(JSON.stringify({ error: 'timeout', lastTick, logs: logs.slice(-5) }))
  try { ws.close() } catch {}
  process.exit(1)
}, timeoutMs)

ws.addEventListener('open', () => {
  ws.send(JSON.stringify({
    type: 'start_arena',
    player_code: playerCode,
    opponent_code: opp.code,
    seed,
    mode,
    friendly_fire: false,
    robot_id: 18,
    player_name: 'Trisha',
    opponent_name: opp.name,
    speed,
    opponent_builtin: opp.id,
  }))
})

ws.addEventListener('message', (ev) => {
  let msg
  try { msg = JSON.parse(ev.data) } catch { return }
  if (msg.type === 'log' && msg.message) {
    logs.push(msg.message)
    if (String(msg.message).includes('execution timeout')) playerError = msg.message
  }
  if (msg.type === 'state' && msg.tick != null) lastTick = msg.tick
  if (msg.type === 'done') {
    done = msg
    clearTimeout(timer)
    const out = {
      mode, seed, opp: args.opp,
      winner: msg.winner,
      message: msg.message,
      steps: msg.playerSteps ?? lastTick,
      timeout: !!(playerError || (msg.message || '').includes('crashed')),
      playerError,
      logsTail: logs.filter((l) => /timeout|stopped|Captured|Eliminated|Max steps/.test(l)).slice(-8),
    }
    console.log(JSON.stringify(out))
    ws.close()
    process.exit(0)
  }
  if (msg.type === 'error') {
    clearTimeout(timer)
    console.log(JSON.stringify({ error: msg.error, lastTick }))
    ws.close()
    process.exit(1)
  }
})

ws.addEventListener('error', (e) => {
  clearTimeout(timer)
  console.error(JSON.stringify({ error: String(e.message || e) }))
  process.exit(1)
})
