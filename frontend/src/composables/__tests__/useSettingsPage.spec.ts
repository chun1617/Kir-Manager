import { describe, it, expect } from 'vitest'
import * as fc from 'fast-check'
import { useSettingsPage } from '../useSettingsPage'
import type { SettingsTab } from '@/types/settings'

describe('Feature: settings-tabbed-layout, useSettingsPage composable', () => {
  describe('Property 1: Tab 切換更新狀態', () => {
    it('should update activeTab when handleTabChange is called', () => {
      fc.assert(
        fc.property(
          fc.constantFrom<SettingsTab>('basic', 'autoSwitch'),
          fc.constantFrom<SettingsTab>('basic', 'autoSwitch'),
          (initialTab, targetTab) => {
            const { activeTab, handleTabChange, resetState } = useSettingsPage()
            
            // 設定初始狀態
            resetState()
            activeTab.value = initialTab
            
            // 執行切換
            handleTabChange(targetTab)
            
            // 驗證：activeTab 應更新為目標 Tab
            return activeTab.value === targetTab
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 2: 頁面重入重置狀態', () => {
    it('should reset activeTab to basic when resetState is called', () => {
      fc.assert(
        fc.property(
          fc.constantFrom<SettingsTab>('basic', 'autoSwitch'),
          (currentTab) => {
            const { activeTab, resetState } = useSettingsPage()
            
            // 設定任意狀態
            activeTab.value = currentTab
            
            // 執行重置
            resetState()
            
            // 驗證：activeTab 應重置為 'basic'
            return activeTab.value === 'basic'
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 6: Tab 禁用狀態與操作狀態同步', () => {
    it('should sync isTabDisabled with isToggling', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          (togglingState) => {
            const { isToggling, isTabDisabled } = useSettingsPage()
            
            // 設定 isToggling 狀態
            isToggling.value = togglingState
            
            // 驗證：isTabDisabled 應與 isToggling 同步
            return isTabDisabled.value === togglingState
          }
        ),
        { numRuns: 100 }
      )
    })

    it('should prevent tab change when isToggling is true', () => {
      const { activeTab, isToggling, handleTabChange, resetState } = useSettingsPage()
      
      resetState()
      expect(activeTab.value).toBe('basic')
      
      // 設定 isToggling 為 true
      isToggling.value = true
      
      // 嘗試切換 Tab
      handleTabChange('autoSwitch')
      
      // 驗證：Tab 不應切換
      expect(activeTab.value).toBe('basic')
    })

    it('should allow tab change when isToggling is false', () => {
      const { activeTab, isToggling, handleTabChange, resetState } = useSettingsPage()
      
      resetState()
      expect(activeTab.value).toBe('basic')
      
      // 確保 isToggling 為 false
      isToggling.value = false
      
      // 執行切換
      handleTabChange('autoSwitch')
      
      // 驗證：Tab 應成功切換
      expect(activeTab.value).toBe('autoSwitch')
    })
  })

  describe('Unit Tests', () => {
    it('should initialize with default values', () => {
      const { activeTab, isTabSwitching, isToggling, isTabDisabled } = useSettingsPage()
      
      expect(activeTab.value).toBe('basic')
      expect(isTabSwitching.value).toBe(false)
      expect(isToggling.value).toBe(false)
      expect(isTabDisabled.value).toBe(false)
    })

    it('should reset all state values', () => {
      const { activeTab, isTabSwitching, isToggling, resetState } = useSettingsPage()
      
      // 修改狀態
      activeTab.value = 'autoSwitch'
      isTabSwitching.value = true
      isToggling.value = true
      
      // 重置
      resetState()
      
      // 驗證
      expect(activeTab.value).toBe('basic')
      expect(isTabSwitching.value).toBe(false)
      expect(isToggling.value).toBe(false)
    })
  })
})
