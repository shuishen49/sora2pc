
export async function streamCompletion(payload, apiKey, baseUrl, { onMessage, onFinish, onError, signal }) {
  const url = `${baseUrl.replace(/\/$/, '')}/v1/chat/completions`

  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`,
        'Accept': 'text/event-stream'
      },
      body: JSON.stringify(payload),
      signal
    })

    if (!response.ok) {
        const text = await response.text()
        let errorMsg = `HTTP ${response.status}`
        try {
            const json = JSON.parse(text)
            if (json.error && json.error.message) errorMsg = json.error.message
            else if (json.message) errorMsg = json.message
        } catch {
            if (text) errorMsg = text
        }
        throw new Error(errorMsg)
    }

    if (!response.body) throw new Error('ReadableStream not supported')

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      const chunk = decoder.decode(value, { stream: true })
      buffer += chunk

      const lines = buffer.split('\n\n')
      // Keep the last partial line in the buffer
      buffer = lines.pop()

      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed.startsWith('data: ')) continue

        const data = trimmed.slice(6)
        if (data === '[DONE]') {
            if (onFinish) onFinish()
            return
        }

        try {
            const parsed = JSON.parse(data)
            if (onMessage) onMessage(parsed)
        } catch (e) {
            // Ignore parse errors for partial JSON, though with \n\n split it should be full
            console.warn('JSON parse error in stream', e)
        }
      }
    }

    if (onFinish) onFinish()

  } catch (err) {
    if (onError) onError(err)
    else throw err
  }
}
