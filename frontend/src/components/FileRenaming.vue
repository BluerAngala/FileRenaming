<script setup>
import { ref, computed, h, watch, onMounted } from 'vue'
import { GetFilesFromPaths, RenameFiles, OpenFileDialog, OpenFolderDialog, GenerateAINames, AIRenameFiles, SetAIAPIKey, SetAIModel, GetAIModels, SaveAIConfig, LoadAIConfig, SavePromptTemplates, LoadPromptTemplates } from '../../wailsjs/go/main/App'
import { useMessage } from 'naive-ui'

const message = useMessage()

// 不再使用localStorage，改用后端文件保存

// 文件列表表格列配置
const fileTableColumns = [
  {
    title: '原文件名',
    key: 'name',
    ellipsis: { tooltip: true }
  },
  {
    title: '新文件名',
    key: 'newName',
    ellipsis: { tooltip: true },
    render: (row) => {
      return row.newName !== row.name
        ? h('span', { style: { color: '#16a34a' } }, row.newName)
        : h('span', { style: { color: '#6b7280' } }, row.newName)
    }
  }
]

// 文件列表
const files = ref([])
const isLoading = ref(false)
const isAIGenerating = ref(false)
const renameMode = ref('rule') // 'rule' 或 'ai'
const renameRule = ref({
  mode: 'rule',
  pattern: '*',
  replaceFrom: '',
  replaceTo: '',
  addPrefix: '',
  addSuffix: '',
  caseType: '',
  numberStart: 1,
  numberStep: 1,
  aiPrompt: '',
  aiGenerated: []
})

// AI相关状态
const aiApiKey = ref('')
const aiBaseURL = ref('https://api.siliconflow.cn/v1')
const aiModel = ref('')
const aiPrompt = ref('')
const aiModelOptions = ref([])
const isLoadingModels = ref(false)

// 提示词模板相关状态
const promptTemplates = ref([])
const showTemplateModal = ref(false)
const newTemplateName = ref('')
const newTemplateContent = ref('')
const editingTemplateIndex = ref(-1)

// 预览新文件名
const previewFiles = computed(() => {
  if (renameMode.value === 'ai') {
    // AI模式：使用AI生成的文件名
    return files.value.map((file, index) => {
      const ext = file.name.substring(file.name.lastIndexOf('.'))
      const aiName = renameRule.value.aiGenerated[index] || ''
      const newName = aiName ? aiName + ext : file.name
      return {
        ...file,
        newName: newName === file.name ? file.name : newName
      }
    })
  } else {
    // 规则模式：使用原有逻辑
    return files.value.map((file, index) => {
      let newName = file.name
      const ext = file.name.substring(file.name.lastIndexOf('.'))
      let nameWithoutExt = file.name.substring(0, file.name.lastIndexOf('.'))

      // 应用替换规则
      if (renameRule.value.replaceFrom) {
        nameWithoutExt = nameWithoutExt.replaceAll(renameRule.value.replaceFrom, renameRule.value.replaceTo)
      }

      // 应用大小写规则
      switch (renameRule.value.caseType) {
        case 'lower':
          nameWithoutExt = nameWithoutExt.toLowerCase()
          break
        case 'upper':
          nameWithoutExt = nameWithoutExt.toUpperCase()
          break
        case 'title':
          nameWithoutExt = nameWithoutExt.charAt(0).toUpperCase() + nameWithoutExt.slice(1).toLowerCase()
          break
      }

      // 添加前缀和后缀
      if (renameRule.value.addPrefix) {
        nameWithoutExt = renameRule.value.addPrefix + nameWithoutExt
      }
      if (renameRule.value.addSuffix) {
        nameWithoutExt = nameWithoutExt + renameRule.value.addSuffix
      }

      // 添加数字编号
      if (renameRule.value.numberStart > 0 || renameRule.value.numberStep > 0) {
        const number = renameRule.value.numberStart + index * (renameRule.value.numberStep || 1)
        nameWithoutExt = `${nameWithoutExt}_${number}`
      }

      newName = nameWithoutExt + ext
      return {
        ...file,
        newName: newName === file.name ? file.name : newName
      }
    })
  }
})

// 处理文件路径列表
const processFilePaths = async (paths, recursive = false) => {
  if (!paths || paths.length === 0) return

  isLoading.value = true
  message.loading(`正在处理 ${paths.length} 个文件/文件夹...`, { duration: 0 })

  try {
    const fileList = await GetFilesFromPaths(paths, recursive)
    files.value = fileList
    // 清空AI生成的结果
    if (renameMode.value === 'ai') {
      renameRule.value.aiGenerated = []
    }

    if (fileList.length > 0) {
      message.destroyAll()
      message.success(`成功获取 ${fileList.length} 个文件！`, { duration: 3000 })
    } else {
      message.destroyAll()
      message.warning('未找到文件', { duration: 3000 })
    }
  } catch (error) {
    message.destroyAll()
    message.error(`获取文件列表失败: ${error}`, { duration: 3000 })
  } finally {
    isLoading.value = false
  }
}

// 文件选择处理 - 使用文件对话框
const handleFileSelect = async () => {
  try {
    const paths = await OpenFileDialog()
    if (paths && paths.length > 0) {
      await processFilePaths(paths)
    }
  } catch (error) {
    message.error(`打开文件对话框失败: ${error}`, { duration: 3000 })
  }
}

// 文件夹选择处理
const handleFolderSelect = async () => {
  try {
    const paths = await OpenFolderDialog()
    if (paths && paths.length > 0) {
      // 选择文件夹时，递归获取文件夹内的所有文件
      await processFilePaths(paths, true)
    }
  } catch (error) {
    message.error(`打开文件夹对话框失败: ${error}`, { duration: 3000 })
  }
}

// 清空文件列表
const clearFiles = () => {
  files.value = []
  if (renameMode.value === 'ai') {
    renameRule.value.aiGenerated = []
  }
}

// 加载模型列表
const loadAIModels = async () => {
  if (!aiApiKey.value.trim()) {
    return
  }

  isLoadingModels.value = true
  try {
    // 获取文本类型的模型（chat模型）
    const models = await GetAIModels('text')
    aiModelOptions.value = models.map(model => ({
      label: model.id,
      value: model.id
    }))
    
    // 如果没有选择模型且列表不为空，默认选择第一个
    if (!aiModel.value && aiModelOptions.value.length > 0) {
      aiModel.value = aiModelOptions.value[0].value
    }
  } catch (error) {
    message.error(`加载模型列表失败: ${error}`, { duration: 3000 })
    aiModelOptions.value = []
  } finally {
    isLoadingModels.value = false
  }
}

// 设置AI API Key
const handleSetAIAPIKey = async () => {
  if (!aiApiKey.value.trim()) {
    message.warning('请输入API密钥', { duration: 3000 })
    return
  }

  try {
    await SetAIAPIKey(aiApiKey.value, aiBaseURL.value)
    // 保存配置（watch会自动保存，但这里确保保存）
    saveAIConfig()
    message.success('API配置保存成功', { duration: 3000 })
    // 加载模型列表
    await loadAIModels()
    // 设置模型
    if (aiModel.value) {
      await SetAIModel(aiModel.value)
    }
  } catch (error) {
    message.error(`设置API配置失败: ${error}`, { duration: 3000 })
  }
}

// 生成AI文件名
const handleGenerateAINames = async () => {
  if (files.value.length === 0) {
    message.warning('请先添加文件', { duration: 3000 })
    return
  }

  if (!aiPrompt.value.trim()) {
    message.warning('请输入重命名需求描述', { duration: 3000 })
    return
  }

  isAIGenerating.value = true
  message.loading('AI正在生成文件名...', { duration: 0 })

  try {
    const newNames = await GenerateAINames(files.value, aiPrompt.value)
    renameRule.value.aiGenerated = newNames
    renameRule.value.aiPrompt = aiPrompt.value
    message.destroyAll()
    message.success('AI生成文件名成功！', { duration: 3000 })
  } catch (error) {
    message.destroyAll()
    message.error(`AI生成失败: ${error}`, { duration: 5000 })
  } finally {
    isAIGenerating.value = false
  }
}

// 执行重命名
const executeRename = async () => {
  if (files.value.length === 0) {
    message.warning('请先添加文件', { duration: 3000 })
    return
  }

  try {
    if (renameMode.value === 'ai') {
      // AI模式
      if (renameRule.value.aiGenerated.length === 0) {
        message.warning('请先生成AI文件名', { duration: 3000 })
        return
      }
      const errors = await AIRenameFiles(files.value, renameRule.value.aiGenerated)
      if (errors && errors.length > 0) {
        message.error(`部分文件重命名失败:\n${errors.join('\n')}`, { duration: 5000 })
      } else {
        message.success('重命名成功!', { duration: 3000 })
        files.value = []
        renameRule.value.aiGenerated = []
        aiPrompt.value = ''
      }
    } else {
      // 规则模式
      renameRule.value.mode = 'rule'
      const errors = await RenameFiles(files.value, renameRule.value)
      if (errors && errors.length > 0) {
        message.error(`部分文件重命名失败:\n${errors.join('\n')}`, { duration: 5000 })
      } else {
        message.success('重命名成功!', { duration: 3000 })
        files.value = []
      }
    }
  } catch (error) {
    message.error(`重命名失败: ${error}`, { duration: 3000 })
  }
}

// 切换模式
const handleModeChange = (mode) => {
  renameMode.value = mode
  renameRule.value.mode = mode
  if (mode === 'ai') {
    renameRule.value.aiGenerated = []
  }
}

// 初始化标志，避免加载时触发watch
let isInitializing = false

// 保存AI配置到后端文件
const saveAIConfig = async () => {
  const config = {
    apiKey: aiApiKey.value,
    baseURL: aiBaseURL.value,
    model: aiModel.value
  }
  try {
    await SaveAIConfig(config)
  } catch (error) {
    console.error('保存配置失败:', error)
  }
}

// 从后端文件加载AI配置
const loadAIConfig = async () => {
  try {
    isInitializing = true
    const config = await LoadAIConfig()
    
    if (config.apiKey) {
      aiApiKey.value = config.apiKey
    }
    if (config.baseURL) {
      aiBaseURL.value = config.baseURL
    }
    if (config.model) {
      aiModel.value = config.model
    }
    
    // 如果有API密钥，自动设置并加载模型列表
    if (config.apiKey) {
      try {
        await SetAIAPIKey(config.apiKey, config.baseURL || 'https://api.siliconflow.cn/v1')
        if (config.model) {
          await SetAIModel(config.model)
        }
        await loadAIModels()
      } catch (err) {
        console.error('加载API配置失败:', err)
      }
    }
  } catch (error) {
    console.error('加载配置失败:', error)
  } finally {
    isInitializing = false
  }
}

// 监听API密钥变化，自动保存和设置
watch(aiApiKey, async (newKey, oldKey) => {
  // 避免初始化时触发
  if (isInitializing) return
  
  if (newKey && newKey.trim() && newKey !== oldKey) {
    // 自动保存配置
    saveAIConfig()
    
    // 自动设置API密钥并加载模型列表
    try {
      await SetAIAPIKey(newKey.trim(), aiBaseURL.value)
      await loadAIModels()
      if (aiModel.value) {
        await SetAIModel(aiModel.value)
      }
    } catch (error) {
      console.error('自动设置API密钥失败:', error)
    }
  } else if (!newKey || !newKey.trim()) {
    // 清空时也保存
    saveAIConfig()
  }
})

// 监听模型变化，自动保存
watch(aiModel, () => {
  if (!isInitializing && aiApiKey.value) {
    saveAIConfig()
    if (aiModel.value) {
      SetAIModel(aiModel.value).catch(err => {
        console.error('设置模型失败:', err)
      })
    }
  }
})

// 加载提示词模板
const loadPromptTemplates = async () => {
  try {
    const result = await LoadPromptTemplates()
    promptTemplates.value = result.templates || []
  } catch (error) {
    console.error('加载模板失败:', error)
    message.error('加载模板失败', { duration: 3000 })
  }
}

// 保存提示词模板
const savePromptTemplates = async () => {
  try {
    await SavePromptTemplates({ templates: promptTemplates.value })
    message.success('模板保存成功', { duration: 2000 })
  } catch (error) {
    console.error('保存模板失败:', error)
    message.error('保存模板失败', { duration: 3000 })
  }
}

// 使用模板 - 使用完整的模板内容，不受显示截断影响
const useTemplate = (template) => {
  // 使用完整的 template.content，确保不会因为显示截断而丢失内容
  aiPrompt.value = template.content
  message.success(`已应用模板：${template.name}`, { duration: 2000 })
}

// 打开新增模板对话框
const openAddTemplateModal = () => {
  editingTemplateIndex.value = -1
  newTemplateName.value = ''
  newTemplateContent.value = ''
  showTemplateModal.value = true
}

// 打开编辑模板对话框
const openEditTemplateModal = (index) => {
  editingTemplateIndex.value = index
  newTemplateName.value = promptTemplates.value[index].name
  newTemplateContent.value = promptTemplates.value[index].content
  showTemplateModal.value = true
}

// 保存模板（新增或编辑）
const handleSaveTemplate = () => {
  if (!newTemplateName.value.trim()) {
    message.warning('请输入模板名称', { duration: 2000 })
    return
  }
  if (!newTemplateContent.value.trim()) {
    message.warning('请输入模板内容', { duration: 2000 })
    return
  }

  const template = {
    name: newTemplateName.value.trim(),
    content: newTemplateContent.value.trim()
  }

  if (editingTemplateIndex.value >= 0) {
    // 编辑模式
    promptTemplates.value[editingTemplateIndex.value] = template
    message.success('模板更新成功', { duration: 2000 })
  } else {
    // 新增模式
    promptTemplates.value.push(template)
    message.success('模板添加成功', { duration: 2000 })
  }

  savePromptTemplates()
  showTemplateModal.value = false
}

// 删除模板
const deleteTemplate = (index) => {
  promptTemplates.value.splice(index, 1)
  savePromptTemplates()
  message.success('模板已删除', { duration: 2000 })
}

// 组件挂载时加载配置
onMounted(() => {
  loadAIConfig()
  loadPromptTemplates()
})
</script>

<template>
  <div class="h-screen text-gray-800 flex flex-col overflow-hidden">
    <h1 class="text-xl font-bold text-center py-2 flex-shrink-0">文件重命名工具</h1>

    <!-- 双栏布局 -->
    <div class="grid grid-cols-2 gap-3 flex-1 p-3 overflow-hidden">
      <!-- 左侧：文件选择和列表 -->
      <div class="space-y-3 flex flex-col overflow-hidden">
        <!-- 文件选择区域 -->
        <div class="flex-shrink-0 bg-gray-100 rounded-lg p-4 flex flex-col items-center justify-center border border-gray-200">
            <div v-if="isLoading" style="font-size: 36px; margin-bottom: 8px" class="animate-spin">⏳</div>
            <div v-else style="font-size: 36px; margin-bottom: 8px">📁</div>
            <n-text style="font-size: 14px; display: block; margin-bottom: 4px">
              选择要重命名的文件或文件夹
            </n-text>
            <n-p depth="3" style="margin: 4px 0 8px 0; font-size: 11px">
              支持选择多个文件或整个文件夹
            </n-p>
            <div class="flex gap-2">
              <n-button type="primary" size="medium" :loading="isLoading" @click="handleFileSelect">
                选择文件
              </n-button>
              <n-button type="default" size="medium" :loading="isLoading" @click="handleFolderSelect">
                选择文件夹
              </n-button>
            </div>
          </div>

          <!-- 文件列表 -->
          <div class="bg-gray-100 rounded-lg p-3 flex flex-col flex-1 overflow-hidden border border-gray-200">
            <div class="flex justify-between items-center mb-2 flex-shrink-0">
              <h2 class="text-lg font-bold">文件列表 ({{ files.length }})</h2>
              <n-button v-if="files.length > 0" type="error" size="small" @click="clearFiles">
                清空列表
              </n-button>
            </div>
            <div v-if="files.length === 0" class="flex-1 flex items-center justify-center text-gray-400">
              <div class="text-center">
                <div class="text-4xl mb-2">📄</div>
                <p>暂无文件</p>
                <p class="text-sm mt-2">拖放文件或点击按钮选择文件</p>
              </div>
            </div>
            <n-data-table v-else :columns="fileTableColumns" :data="previewFiles" class="flex-1" />
          </div>
        </div>

        <!-- 右侧：重命名规则配置 -->
        <div class="space-y-3 flex flex-col overflow-hidden">
          <!-- 模式切换 -->
          <div class="bg-gray-100 rounded-lg p-3 flex-shrink-0 border border-gray-200">
            <n-radio-group v-model:value="renameMode" @update:value="handleModeChange">
              <n-space>
                <n-radio value="rule">规则模式</n-radio>
                <n-radio value="ai">AI模式</n-radio>
              </n-space>
            </n-radio-group>
          </div>

          <!-- 规则模式配置 -->
          <div v-if="renameMode === 'rule'" class="bg-gray-100 rounded-lg flex-1 overflow-y-auto p-3 scrollbar-hide border border-gray-200">
            <n-form :model="renameRule" label-placement="top" label-width="auto" class="space-y-2">
              <n-form-item label="文件匹配模式">
                <n-input v-model:value="renameRule.pattern" placeholder="如: *.txt, *.jpg, *" />
              </n-form-item>
              <n-form-item label="大小写转换">
                <n-select v-model:value="renameRule.caseType" :options="[
                  { label: '不转换', value: '' },
                  { label: '小写', value: 'lower' },
                  { label: '大写', value: 'upper' },
                  { label: '首字母大写', value: 'title' }
                ]" placeholder="请选择大小写转换方式" />
              </n-form-item>
              <div class="grid grid-cols-2 gap-2">
                <n-form-item label="替换 (从)">
                  <n-input v-model:value="renameRule.replaceFrom" placeholder="要替换的文本" />
                </n-form-item>
                <n-form-item label="替换 (到)">
                  <n-input v-model:value="renameRule.replaceTo" placeholder="替换为" />
                </n-form-item>
              </div>
              <div class="grid grid-cols-2 gap-2">
                <n-form-item label="添加前缀">
                  <n-input v-model:value="renameRule.addPrefix" placeholder="前缀文本" />
                </n-form-item>
                <n-form-item label="添加后缀">
                  <n-input v-model:value="renameRule.addSuffix" placeholder="后缀文本" />
                </n-form-item>
              </div>
              <div class="grid grid-cols-2 gap-2">
                <n-form-item label="数字起始值">
                  <n-input-number v-model:value="renameRule.numberStart" :min="0" class="w-full text-center"
                    button-placement="both" />
                </n-form-item>
                <n-form-item label="数字步长">
                  <n-input-number v-model:value="renameRule.numberStep" :min="1" class="w-full text-center"
                    button-placement="both" />
                </n-form-item>
              </div>
            </n-form>
          </div>

          <!-- AI模式配置 -->
          <div v-if="renameMode === 'ai'" class="bg-gray-100 rounded-lg flex-1 overflow-y-auto p-3 space-y-3 scrollbar-hide border border-gray-200">
            
            <!-- API配置 - 可折叠 -->
            <n-collapse>
              <n-collapse-item name="api-config">
                <template #header>
                  <div class="flex items-center justify-between w-full pr-2">
                    <span>API配置（使用前请先配置）</span>
                    <n-button 
                      type="info" 
                      size="small" 
                      @click.stop="() => window.open('https://cloud.siliconflow.cn/i/WFoChvZf', '_blank')"
                    >
                      免费AI密钥
                    </n-button>
                  </div>
                </template>
                <n-form label-placement="top" class="space-y-2">
                  <n-form-item label="API密钥">
                    <n-input v-model:value="aiApiKey" type="password" placeholder="请输入SiliconFlow API密钥" show-password-on="click" />
                  </n-form-item>
                  <n-form-item label="模型选择">
                    <n-select 
                      v-model:value="aiModel" 
                      :options="aiModelOptions" 
                      placeholder="选择AI模型（保存API密钥后自动加载）" 
                      filterable
                      :loading="isLoadingModels"
                      :disabled="aiModelOptions.length === 0"
                      @update:value="(value) => { if (value) SetAIModel(value) }"
                    />
                    <div v-if="aiApiKey && aiModelOptions.length === 0 && !isLoadingModels" style="margin-top: 8px">
                      <n-button 
                        text 
                        size="small" 
                        @click="loadAIModels"
                      >
                        重新加载模型列表
                      </n-button>
                    </div>
                  </n-form-item>
                  <n-button type="primary" block @click="handleSetAIAPIKey">保存API配置</n-button>
                </n-form>
              </n-collapse-item>
            </n-collapse>

            <!-- AI提示词 -->
            <n-form-item label="重命名需求描述">
              <n-input
                v-model:value="aiPrompt"
                type="textarea"
                :rows="3"
                class="text-left"
                placeholder="例如：按照今天日期+源文件名称+序号命名"
              />
            </n-form-item>

            <!-- 提示词模板 -->
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium">提示词模板</span>
                <n-button size="small" type="primary" @click="openAddTemplateModal">
                  新增模板
                </n-button>
              </div>
              <div v-if="promptTemplates.length === 0" class="text-center text-gray-400 text-xs py-4">
                暂无模板，点击"新增模板"创建
              </div>
              <div v-else class="grid grid-cols-2 gap-2 max-h-40 overflow-y-auto scrollbar-hide">
                <div
                  v-for="(template, index) in promptTemplates"
                  :key="index"
                  class="bg-white rounded border border-gray-200 p-2 hover:border-blue-400 cursor-pointer transition-colors group relative min-h-[60px]"
                  @click="useTemplate(template)"
                >
                  <div class="pr-12">
                    <div class="text-xs font-medium text-gray-700 mb-1 truncate text-left">{{ template.name }}</div>
                    <div class="text-xs text-gray-500 text-left break-words line-clamp-2 leading-relaxed">{{ template.content }}</div>
                  </div>
                  <div class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 flex gap-1 z-10">
                    <n-button
                      size="tiny"
                      type="info"
                      text
                      @click.stop="openEditTemplateModal(index)"
                    >
                      编辑
                    </n-button>
                    <n-button
                      size="tiny"
                      type="error"
                      text
                      @click.stop="deleteTemplate(index)"
                    >
                      删除
                    </n-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- AI生成结果提示 -->
            <n-alert v-if="renameRule.aiGenerated.length > 0" type="success" title="生成成功">
              已为 {{ renameRule.aiGenerated.length }} 个文件生成新文件名，请查看左侧预览
            </n-alert>
          </div>

          <!-- 操作按钮 -->
          <div v-if="files.length > 0" class="bg-gray-100 rounded-lg p-3 flex-shrink-0 border border-gray-200">
            <div v-if="renameMode === 'ai'" class="flex gap-2">
              <n-button
                type="primary"
                size="medium"
                :loading="isAIGenerating"
                :disabled="files.length === 0 || !aiPrompt.trim()"
                @click="handleGenerateAINames"
                class="flex-1"
              >
                {{ isAIGenerating ? 'AI生成中...' : '生成AI文件名' }}
              </n-button>
              <n-button
                type="primary"
                size="medium"
                :disabled="renameRule.aiGenerated.length === 0"
                @click="executeRename"
                class="flex-1"
              >
                执行重命名
              </n-button>
            </div>
            <n-button
              v-else
              type="primary"
              size="medium"
              block
              @click="executeRename"
            >
              执行重命名
            </n-button>
          </div>
        </div>
      </div>
  </div>

  <!-- 新增/编辑模板对话框 -->
  <n-modal v-model:show="showTemplateModal" preset="dialog" :title="editingTemplateIndex >= 0 ? '编辑模板' : '新增模板'">
    <div class="space-y-3">
      <n-form-item label="模板名称">
        <n-input v-model:value="newTemplateName" placeholder="请输入模板名称" />
      </n-form-item>
      <n-form-item label="模板内容">
        <n-input
          v-model:value="newTemplateContent"
          type="textarea"
          :rows="4"
          placeholder="请输入模板内容"
        />
      </n-form-item>
    </div>
    <template #action>
      <n-space>
        <n-button @click="showTemplateModal = false">取消</n-button>
        <n-button type="primary" @click="handleSaveTemplate">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
/* 隐藏滚动条但保持滚动功能 */
.scrollbar-hide {
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
}

.scrollbar-hide::-webkit-scrollbar {
  display: none; /* Chrome, Safari, Opera */
}

/* 文本截断样式 */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
}
</style>
