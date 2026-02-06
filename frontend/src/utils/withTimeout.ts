/**
 * 超時錯誤類別
 * @description 當操作超過指定時間未完成時拋出
 * @requirements 8.4 - 操作超時保護
 */
export class TimeoutError extends Error {
  constructor(message: string = '操作超時') {
    super(message)
    this.name = 'TimeoutError'
  }
}

/**
 * 為 Promise 添加超時保護
 * @description 如果操作在指定時間內未完成，將拋出 TimeoutError
 * @requirements 8.4 - 操作超時保護
 * @param promise 要執行的 Promise
 * @param timeoutMs 超時時間（毫秒）
 * @param message 超時錯誤訊息
 * @returns Promise<T> - 原始 Promise 的結果或超時錯誤
 */
export async function withTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  message?: string
): Promise<T> {
  // 處理 0 或負數超時時間 - 立即超時
  if (timeoutMs <= 0) {
    throw new TimeoutError(message || '操作超時')
  }

  let timeoutId: ReturnType<typeof setTimeout> | null = null

  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutId = setTimeout(() => {
      reject(new TimeoutError(message || '操作超時'))
    }, timeoutMs)
  })

  try {
    const result = await Promise.race([promise, timeoutPromise])
    return result
  } finally {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
    }
  }
}
