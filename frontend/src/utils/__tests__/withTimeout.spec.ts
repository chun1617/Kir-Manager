import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { withTimeout, TimeoutError } from '../withTimeout'

describe('Feature: app-vue-decoupling, withTimeout utility', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  // ============================================================================
  // Property 33: 操作超時保護
  // Validates: Requirements 8.4
  // ============================================================================

  describe('Property 33: 操作超時保護', () => {
    it('正常完成的操作應返回結果', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.anything(),
          fc.integer({ min: 100, max: 1000 }),
          async (expectedResult, timeoutMs) => {
            // 建立一個立即完成的 Promise
            const operation = Promise.resolve(expectedResult)
            
            const result = await withTimeout(operation, timeoutMs)
            
            return result === expectedResult
          }
        ),
        { numRuns: 50 }
      )
    })

    it('超時的操作應拋出 TimeoutError', async () => {
      const slowOperation = new Promise((resolve) => {
        setTimeout(() => resolve('completed'), 5000)
      })
      
      const promise = withTimeout(slowOperation, 1000)
      
      // 快進超過超時時間
      vi.advanceTimersByTime(1001)
      
      await expect(promise).rejects.toThrow(TimeoutError)
    })

    it('超時錯誤應包含自定義訊息', async () => {
      const slowOperation = new Promise((resolve) => {
        setTimeout(() => resolve('completed'), 5000)
      })
      
      const customMessage = '操作超時，請稍後重試'
      const promise = withTimeout(slowOperation, 1000, customMessage)
      
      vi.advanceTimersByTime(1001)
      
      await expect(promise).rejects.toThrow(customMessage)
    })

    it('操作在超時前完成應取消超時計時器', async () => {
      const fastOperation = Promise.resolve('fast result')
      
      const result = await withTimeout(fastOperation, 5000)
      
      expect(result).toBe('fast result')
      
      // 快進超過超時時間，不應有任何錯誤
      vi.advanceTimersByTime(6000)
    })

    it('操作拋出的錯誤應正常傳播', async () => {
      const errorMessage = 'Operation failed'
      const failingOperation = Promise.reject(new Error(errorMessage))
      
      await expect(withTimeout(failingOperation, 5000)).rejects.toThrow(errorMessage)
    })

    it('超時時間為 0 應立即超時', async () => {
      const operation = new Promise((resolve) => {
        setTimeout(() => resolve('completed'), 100)
      })
      
      const promise = withTimeout(operation, 0)
      
      vi.advanceTimersByTime(1)
      
      await expect(promise).rejects.toThrow(TimeoutError)
    })

    it('負數超時時間應立即超時', async () => {
      const operation = new Promise((resolve) => {
        setTimeout(() => resolve('completed'), 100)
      })
      
      const promise = withTimeout(operation, -100)
      
      vi.advanceTimersByTime(1)
      
      await expect(promise).rejects.toThrow(TimeoutError)
    })

    it('TimeoutError 應是 Error 的實例', () => {
      const error = new TimeoutError('test')
      expect(error).toBeInstanceOf(Error)
      expect(error.name).toBe('TimeoutError')
    })

    it('withTimeout 應保持 Promise 的類型', async () => {
      interface TestResult {
        id: number
        name: string
      }
      
      const typedOperation: Promise<TestResult> = Promise.resolve({
        id: 1,
        name: 'test'
      })
      
      const result = await withTimeout(typedOperation, 1000)
      
      expect(result.id).toBe(1)
      expect(result.name).toBe('test')
    })
  })

  describe('邊界情況', () => {
    it('非常大的超時值應正常工作', async () => {
      const operation = Promise.resolve('result')
      const result = await withTimeout(operation, Number.MAX_SAFE_INTEGER)
      expect(result).toBe('result')
    })

    it('void 返回類型的操作應正常工作', async () => {
      const voidOperation: Promise<void> = Promise.resolve()
      const result = await withTimeout(voidOperation, 1000)
      expect(result).toBeUndefined()
    })

    it('null 返回值應正常傳遞', async () => {
      const nullOperation: Promise<null> = Promise.resolve(null)
      const result = await withTimeout(nullOperation, 1000)
      expect(result).toBeNull()
    })

    it('undefined 返回值應正常傳遞', async () => {
      const undefinedOperation: Promise<undefined> = Promise.resolve(undefined)
      const result = await withTimeout(undefinedOperation, 1000)
      expect(result).toBeUndefined()
    })
  })

  describe('預設超時訊息', () => {
    it('未提供自定義訊息時應使用預設訊息', async () => {
      const slowOperation = new Promise((resolve) => {
        setTimeout(() => resolve('completed'), 5000)
      })
      
      const promise = withTimeout(slowOperation, 1000)
      
      vi.advanceTimersByTime(1001)
      
      try {
        await promise
        expect.fail('Should have thrown')
      } catch (error) {
        expect(error).toBeInstanceOf(TimeoutError)
        expect((error as TimeoutError).message).toBeTruthy()
      }
    })
  })
})
