package repository

// pixelWriteOpLogScript runs on one cluster slot: XADD op-log, compare stream id / HLC vs pixmeta,
// conditionally RPUSH canvas list + HSET meta. Mirrors domain/crdt DecideLWW ordering (ADR-001).
//
// KEYS[1]=opstream:{y:x} KEYS[2]=pixel:{y:x} KEYS[3]=pixmeta:{y:x}
// ARGV[1]=payload ARGV[2]=writer ARGV[3]=incoming_hlc_ms ARGV[4]=incoming_hlc_lc
//
// Returns: { new_stream_id, replace (0|1), path (1=stream_id,2=hlc_fallback) }
const pixelWriteOpLogScript = `
local opk, listk, metak = KEYS[1], KEYS[2], KEYS[3]
local data, writer, inhms, inhlc = ARGV[1], ARGV[2], ARGV[3], ARGV[4]

local new_id = redis.call(
  'XADD', opk, 'MAXLEN', '~', '1000', '*',
  'data', data, 'writer', writer, 'hlc_ms', inhms, 'hlc_lc', inhlc
)

local sid = redis.call('HGET', metak, 'latest_stream_id')
local sms = redis.call('HGET', metak, 'hlc_ms')
local slc = redis.call('HGET', metak, 'hlc_lc')
if not sms then sms = '0' end
if not slc then slc = '0' end

local replace = 0
local path = 2

local function parse_sid(s)
  if not s or s == '' then return nil, nil end
  local ms, seq = string.match(s, '^(%d+)%-(%d+)$')
  if not ms then return nil, nil end
  return tonumber(ms), tonumber(seq)
end

local nms, nseq = parse_sid(new_id)
local oms, oseq = parse_sid(sid)

if (not sid) or sid == '' then
  replace = 1
  path = 1
elseif nms and oms then
  path = 1
  if nms > oms or (nms == oms and nseq > oseq) then
    replace = 1
  end
else
  path = 2
  local im = tonumber(inhms) or 0
  local il = tonumber(inhlc) or 0
  local sm = tonumber(sms) or 0
  local sl = tonumber(slc) or 0
  if im > sm or (im == sm and il > sl) then
    replace = 1
  end
end

if replace == 1 then
  redis.call('HSET', metak, 'latest_stream_id', new_id, 'hlc_ms', inhms, 'hlc_lc', inhlc)
  redis.call('RPUSH', listk, data)
end

return { new_id, replace, path }
`
