<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import {
  StartSocialLogin,
  StartIdCLogin,
  CreateSnapshotFromOAuth,
  ValidateSnapshotName,
  IsDeepLinkSupported
} from '../../wailsjs/go/main/App'

const { t } = useI18n()

// 定義 emit
const emit = defineEmits<{
  (e: 'snapshotCreated'): void
}>()

// OAuth 登入結果類型
interface OAuthLoginResult {
  success: boolean
  message: string
  accessToken?: string
  refreshToken?: string
  expiresAt?: string
  provider?: string
  authMethod?: string
  clientId?: string
  clientSecret?: string
  clientIdHash?: string
  userCode?: string
  verificationUri?: string
}

// 登入提供者配置
const providers = [
  { id: 'Github', name: 'GitHub', icon: 'Github', method: 'social' },
  { id: 'Google', name: 'Google', icon: 'Google', method: 'social' },
  { id: 'BuilderId', name: 'AWS Builder ID', icon: 'AWS', method: 'idc' }
] as const

// 狀態
const loading = ref<string | null>(null) // 正在登入的 provider ID
const error = ref<string | null>(null)
const loginResult = ref<OAuthLoginResult | null>(null)
const successMessage = ref<string | null>(null) // 成功訊息
const deepLinkSupported = ref<boolean>(true) // 預設為 true，載入後更新

// 快照命名對話框狀態
const showSnapshotDialog = ref(false)
const snapshotName = ref('')
const snapshotNameError = ref<string | null>(null)
const creatingSnapshot = ref(false)

// IdC 登入狀態顯示
const idcUserCode = ref<string | null>(null)
const idcVerificationUri = ref<string | null>(null)

// 計算屬性：是否正在登入
const isLoading = computed(() => loading.value !== null)

// 生命週期：檢查 Deep Link 支援
onMounted(async () => {
  try {
    deepLinkSupported.value = await IsDeepLinkSupported()
  } catch (e) {
    console.error('Failed to check deep link support:', e)
    deepLinkSupported.value = false
  }
})

// 開始登入
const startLogin = async (providerId: string, method: string) => {
  // 重置狀態
  error.value = null
  loading.value = providerId
  idcUserCode.value = null
  idcVerificationUri.value = null

  try {
    let result: OAuthLoginResult

    if (method === 'social') {
      // Social 登入 (GitHub/Google)
      result = await StartSocialLogin(providerId)
    } else {
      // IdC 登入 (AWS Builder ID)
      result = await StartIdCLogin()
      
      // 如果返回了 userCode，顯示給用戶
      if (result.userCode) {
        idcUserCode.value = result.userCode
        idcVerificationUri.value = result.verificationUri || null
      }
    }

    if (result.success) {
      // 登入成功，保存結果並顯示命名對話框
      loginResult.value = result
      showSnapshotDialog.value = true
      snapshotName.value = ''
      snapshotNameError.value = null
    } else {
      // 登入失敗
      error.value = result.message
    }
  } catch (e) {
    error.value = t('oauth.networkError')
    console.error('OAuth login error:', e)
  } finally {
    loading.value = null
    idcUserCode.value = null
    idcVerificationUri.value = null
  }
}

// 驗證快照名稱
const validateName = async (name: string): Promise<string | null> => {
  if (!name.trim()) {
    return t('oauth.snapshotNameEmpty')
  }
  
  try {
    const result = await ValidateSnapshotName(name.trim())
    if (!result.success) {
      return result.message
    }
    return null
  } catch (e) {
    return t('oauth.validationError')
  }
}

// 建立快照
const createSnapshot = async () => {
  if (!loginResult.value || creatingSnapshot.value) return

  const name = snapshotName.value.trim()
  
  // 驗證名稱
  const validationError = await validateName(name)
  if (validationError) {
    snapshotNameError.value = validationError
    return
  }

  creatingSnapshot.value = true
  snapshotNameError.value = null

  try {
    const result = await CreateSnapshotFromOAuth(name, loginResult.value)
    
    if (result.success) {
      // 成功建立快照
      showSnapshotDialog.value = false
      loginResult.value = null
      snapshotName.value = ''
      // 顯示成功訊息
      successMessage.value = t('oauth.snapshotCreated')
      setTimeout(() => { successMessage.value = null }, 3000)
      // 通知父組件刷新列表
      emit('snapshotCreated')
    } else {
      snapshotNameError.value = result.message
    }
  } catch (e) {
    snapshotNameError.value = t('oauth.createSnapshotError')
    console.error('Create snapshot error:', e)
  } finally {
    creatingSnapshot.value = false
  }
}

// 取消建立快照
const cancelSnapshot = () => {
  showSnapshotDialog.value = false
  loginResult.value = null
  snapshotName.value = ''
  snapshotNameError.value = null
  // 顯示取消訊息
  successMessage.value = t('oauth.snapshotCancelled')
  setTimeout(() => { successMessage.value = null }, 3000)
}

// 清除錯誤
const clearError = () => {
  error.value = null
}
</script>

<template>
  <div class="space-y-6">
    <!-- 標題 -->
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-medium text-zinc-200 flex items-center">
        <Icon name="Key" class="w-5 h-5 mr-2 text-app-accent" />
        {{ t('oauth.title') }}
      </h2>
    </div>

    <!-- 說明文字 -->
    <p class="text-zinc-400 text-sm">
      {{ t('oauth.description') }}
    </p>

    <!-- 成功訊息 -->
    <div 
      v-if="successMessage" 
      class="bg-green-900/20 border border-green-800/50 rounded-lg p-4 flex items-start gap-3"
    >
      <Icon name="CheckCircle" class="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="text-green-400 text-sm">{{ successMessage }}</p>
      </div>
    </div>

    <!-- 錯誤訊息 -->
    <div 
      v-if="error" 
      class="bg-red-900/20 border border-red-800/50 rounded-lg p-4 flex items-start gap-3"
    >
      <Icon name="XCircle" class="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="text-red-400 text-sm">{{ error }}</p>
      </div>
      <button 
        @click="clearError"
        class="text-red-400 hover:text-red-300 transition-colors"
      >
        <Icon name="X" class="w-4 h-4" />
      </button>
    </div>

    <!-- IdC 用戶碼顯示 -->
    <div 
      v-if="idcUserCode" 
      class="bg-app-accent/10 border border-app-accent/30 rounded-lg p-4"
    >
      <div class="flex items-center gap-2 mb-2">
        <Icon name="Info" class="w-5 h-5 text-app-accent" />
        <span class="text-zinc-200 font-medium">{{ t('oauth.idcUserCodeTitle') }}</span>
      </div>
      <p class="text-zinc-400 text-sm mb-3">{{ t('oauth.idcUserCodeDesc') }}</p>
      <div class="bg-zinc-900 rounded-lg p-4 text-center">
        <span class="text-2xl font-mono font-bold text-app-accent tracking-wider">
          {{ idcUserCode }}
        </span>
      </div>
      <p class="text-zinc-500 text-xs mt-2 text-center">
        {{ t('oauth.idcWaiting') }}
      </p>
    </div>

    <!-- 登入選項卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div 
        v-for="provider in providers" 
        :key="provider.id"
        class="bg-zinc-900 border border-app-border rounded-xl p-6 flex flex-col items-center gap-4 hover:border-zinc-600 transition-colors"
      >
        <!-- 圖標 -->
        <div class="w-16 h-16 rounded-full bg-zinc-800 flex items-center justify-center">
          <Icon :name="provider.icon" class="w-8 h-8 text-zinc-300" />
        </div>
        
        <!-- 名稱 -->
        <h3 class="text-zinc-200 font-medium">{{ provider.name }}</h3>
        
        <!-- 不支援提示 (僅 Social 登入) -->
        <p 
          v-if="provider.method === 'social' && !deepLinkSupported" 
          class="text-amber-400 text-xs text-center"
        >
          {{ t('oauth.socialNotSupported') }}
        </p>
        
        <!-- 登入按鈕 -->
        <button
          @click="startLogin(provider.id, provider.method)"
          :disabled="isLoading || (provider.method === 'social' && !deepLinkSupported)"
          :class="[
            'w-full py-2.5 px-4 rounded-lg text-sm font-medium transition-all flex items-center justify-center gap-2',
            loading === provider.id
              ? 'bg-app-accent/50 text-white cursor-wait'
              : (isLoading || (provider.method === 'social' && !deepLinkSupported))
                ? 'bg-zinc-800 text-zinc-500 cursor-not-allowed'
                : 'bg-app-accent hover:bg-app-accent/80 text-white'
          ]"
        >
          <Icon 
            v-if="loading === provider.id" 
            name="Loader" 
            class="w-4 h-4 animate-spin" 
          />
          <span>{{ t('oauth.login') }}</span>
        </button>
      </div>
    </div>

    <!-- 快照命名對話框 -->
    <Teleport to="body">
      <div 
        v-if="showSnapshotDialog" 
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <!-- 背景遮罩 -->
        <div 
          class="absolute inset-0 bg-black/60 backdrop-blur-sm"
          @click="cancelSnapshot"
        ></div>
        
        <!-- 對話框 -->
        <div class="relative bg-zinc-900 border border-app-border rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl">
          <!-- 標題 -->
          <h3 class="text-lg font-medium text-zinc-200 mb-2 flex items-center">
            <Icon name="Save" class="w-5 h-5 mr-2 text-app-accent" />
            {{ t('oauth.snapshotDialogTitle') }}
          </h3>
          
          <!-- 說明 -->
          <p class="text-zinc-400 text-sm mb-4">
            {{ t('oauth.snapshotDialogDesc') }}
          </p>
          
          <!-- 輸入框 -->
          <div class="mb-4">
            <label class="block text-zinc-400 text-sm mb-2">
              {{ t('oauth.snapshotNameLabel') }}
            </label>
            <input
              v-model="snapshotName"
              type="text"
              :placeholder="t('oauth.snapshotNamePlaceholder')"
              :disabled="creatingSnapshot"
              @keyup.enter="createSnapshot"
              class="w-full px-3 py-2.5 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-200 text-sm focus:outline-none focus:border-app-accent transition-colors placeholder-zinc-600"
              :class="{ 'border-red-500': snapshotNameError }"
            />
            <!-- 錯誤訊息 -->
            <p v-if="snapshotNameError" class="text-red-400 text-xs mt-2">
              {{ snapshotNameError }}
            </p>
          </div>
          
          <!-- 按鈕 -->
          <div class="flex gap-3">
            <button
              @click="cancelSnapshot"
              :disabled="creatingSnapshot"
              class="flex-1 py-2.5 px-4 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors disabled:opacity-50"
            >
              {{ t('oauth.cancel') }}
            </button>
            <button
              @click="createSnapshot"
              :disabled="creatingSnapshot || !snapshotName.trim()"
              class="flex-1 py-2.5 px-4 bg-app-accent hover:bg-app-accent/80 text-white rounded-lg text-sm transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              <Icon 
                v-if="creatingSnapshot" 
                name="Loader" 
                class="w-4 h-4 animate-spin" 
              />
              <span>{{ t('oauth.create') }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
