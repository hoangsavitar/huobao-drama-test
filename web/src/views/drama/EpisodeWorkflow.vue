<template>
  <div class="page-container">
    <div class="content-wrapper animate-fade-in">
      <AppHeader :fixed="false" :show-logo="false">
        <template #left>
          <el-button text @click="goBack" class="back-btn">
            <el-icon><ArrowLeft /></el-icon>
            <span>{{ $t("workflow.backToProject") }}</span>
          </el-button>
          <h1 class="header-title">
            {{ $t("workflow.episodeProduction", { number: episodeNumber }) }}
          </h1>
        </template>
        <template #center>
          <div class="custom-steps">
            <div
              class="step-item"
              :class="{ active: currentStep >= 0, current: currentStep === 0 }"
              @click="goToStep(0)"
            >
              <div class="step-circle">1</div>
              <span class="step-text">{{ $t("workflow.steps.content") }}</span>
            </div>
            <el-icon class="step-arrow"><ArrowRight /></el-icon>
            <div
              class="step-item"
              :class="{ active: currentStep >= 1, current: currentStep === 1 }"
              @click="goToStep(1)"
            >
              <div class="step-circle">2</div>
              <span class="step-text">{{
                $t("workflow.steps.generateImages")
              }}</span>
            </div>
            <el-icon class="step-arrow"><ArrowRight /></el-icon>
            <div
              class="step-item"
              :class="{ active: currentStep >= 2, current: currentStep === 2 }"
              @click="goToStep(2)"
            >
              <div class="step-circle">3</div>
              <span class="step-text">{{
                $t("workflow.steps.splitStoryboard")
              }}</span>
            </div>
          </div>
        </template>
        <template #right>
          <el-button
            :icon="Setting"
            @click="showModelConfigDialog"
            :title="$t('workflow.modelConfig')"
          >
            Text/Image Config
          </el-button>
        </template>
      </AppHeader>

      <div class="content-container">
        <!-- 阶段 0: 章节内容 + 提取角色场景 -->
        <el-card
          v-show="currentStep === 0"
          shadow="never"
          class="stage-card stage-card-fullscreen"
        >
          <div class="stage-body stage-body-fullscreen">
            <!-- 未保存时显示输入框 -->
            <div v-if="!hasScript || isEditingScript" class="generation-form">
              <el-alert
                v-if="currentEpisode?.narrative_node_id"
                type="info"
                :closable="false"
                show-icon
                style="margin-bottom: 16px"
              >
                <template #title>
                  This episode is a narrative skeleton. Run Agent 2 and Agent 3 from Story generator to fill Episode Content, then generate images manually here.
                </template>
              </el-alert>
              <el-input
                v-model="scriptContent"
                type="textarea"
                :placeholder="$t('workflow.scriptPlaceholder')"
                class="script-textarea script-textarea-fullscreen"
              />

              <div class="action-buttons-inline" style="display: flex; gap: 12px; margin-top: 12px;">
                <el-button
                  type="primary"
                  size="default"
                  @click="saveChapterScript"
                  :disabled="!scriptContent.trim() || generatingScript"
                >
                  <el-icon><Check /></el-icon>
                  <span>{{ $t("workflow.saveChapter") }}</span>
                </el-button>
                <el-button
                  v-if="isEditingScript"
                  type="info"
                  size="default"
                  @click="isEditingScript = false"
                >
                  Cancel
                </el-button>
              </div>
            </div>

            <!-- 已保存时显示内容 -->
            <div v-if="hasScript && !isEditingScript" class="overview-section">
              <div class="episode-info">
                <h3>
                  {{ $t("workflow.chapterContent", { number: episodeNumber }) }}
                </h3>
                <el-tag type="success" size="large">{{
                  $t("workflow.saved")
                }}</el-tag>
              </div>
              <div class="overview-content">
                <el-input
                  v-model="currentEpisode.script_content"
                  type="textarea"
                  :rows="15"
                  readonly
                  class="script-display"
                />
              </div>

              <!-- Action buttons for Edit and Extract -->
              <div style="margin-top: 16px; display: flex; gap: 12px; margin-bottom: 16px;">
                <el-button
                  type="warning"
                  @click="editCurrentEpisodeScript"
                >
                  <el-icon><Edit /></el-icon>
                  <span>Edit Script</span>
                </el-button>
                <el-button
                  type="primary"
                  :loading="extractingCharactersAndBackgrounds"
                  @click="handleExtractCharactersAndBackgrounds"
                >
                  <el-icon><MagicStick /></el-icon>
                  <span>Extract Characters & Scenes</span>
                </el-button>
              </div>

              <!-- Graph Flow Navigation -->
              <div class="graph-navigation-strip" style="margin-bottom: 24px; background: var(--bg-secondary); padding: 14px 18px; border-radius: 8px; border: 1px solid var(--border-primary); display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 16px;">
                <div style="display: flex; align-items: center; gap: 10px;">
                  <span style="font-size: 11px; color: var(--text-muted); font-weight: 700; letter-spacing: 0.8px; text-transform: uppercase;">← FROM</span>
                  <div style="display: flex; gap: 6px; flex-wrap: wrap;">
                    <button
                      v-for="prev in previousEpisodes"
                      :key="prev.id"
                      @click="goToEpisode(prev.episode_number)"
                      style="background: rgba(100,116,139,0.18); border: 1px solid rgba(100,116,139,0.35); color: #94a3b8; border-radius: 6px; padding: 4px 12px; font-size: 12px; font-weight: 600; cursor: pointer; transition: all 0.15s; font-family: inherit;"
                      @mouseover="$event.currentTarget.style.background='rgba(100,116,139,0.32)'"
                      @mouseout="$event.currentTarget.style.background='rgba(100,116,139,0.18)'"
                    >
                      Ep.{{ prev.episode_number }} {{ prev.narrative_node_id ? '('+prev.narrative_node_id+')' : '' }}
                    </button>
                    <span v-if="!previousEpisodes.length" style="font-size: 12px; color: var(--text-muted); font-style: italic; padding: 4px 0;">Entry Episode</span>
                  </div>
                </div>

                <div style="display: flex; align-items: center; gap: 10px;">
                  <span style="font-size: 11px; color: var(--text-muted); font-weight: 700; letter-spacing: 0.8px; text-transform: uppercase;">→ TO</span>
                  <div style="display: flex; gap: 6px; flex-wrap: wrap;">
                    <button
                      v-for="next in nextEpisodes"
                      :key="next.id"
                      @click="goToEpisode(next.episode_number)"
                      style="background: rgba(99,102,241,0.18); border: 1px solid rgba(99,102,241,0.4); color: #a5b4fc; border-radius: 6px; padding: 4px 12px; font-size: 12px; font-weight: 600; cursor: pointer; transition: all 0.15s; font-family: inherit;"
                      @mouseover="$event.currentTarget.style.background='rgba(99,102,241,0.32)'"
                      @mouseout="$event.currentTarget.style.background='rgba(99,102,241,0.18)'"
                    >
                      Ep.{{ next.episode_number }} {{ next.narrative_node_id ? '('+next.narrative_node_id+')' : '' }}
                    </button>
                    <span v-if="!nextEpisodes.length" style="font-size: 12px; color: var(--text-muted); font-style: italic; padding: 4px 0;">End of Branch</span>
                  </div>
                </div>

                <!-- Edit connections trigger -->
                <button
                  @click="openEditGraphDialog"
                  style="background: rgba(99,102,241,0.15); border: 1px solid rgba(99,102,241,0.4); color: #a5b4fc; border-radius: 6px; padding: 5px 14px; font-size: 12px; font-weight: 700; cursor: pointer; display: flex; align-items: center; gap: 6px; font-family: inherit; transition: all 0.15s;"
                  @mouseover="$event.currentTarget.style.background='rgba(99,102,241,0.28)'"
                  @mouseout="$event.currentTarget.style.background='rgba(99,102,241,0.15)'"
                >
                  ⚙ Edit Connections
                </button>
              </div>

              <!-- Metadata Section (Micro-beats, Plot Outline & State) -->
              <div v-if="currentEpisode.description || currentEpisode.state_snapshot" class="episode-metadata" style="margin-top: 16px; margin-bottom: 16px;">
                <el-collapse>
                  <!-- Plot Outline (Agent 1 Architect) -->
                  <el-collapse-item v-if="currentEpisode.state_snapshot?.plot_summary || (!currentEpisode.state_snapshot?.timeline && currentEpisode.description)" title="Plot Outline (Agent 1 Architect)" name="plot">
                    <div style="white-space: pre-wrap; font-size: 13px; color: var(--text-secondary); line-height: 1.6; background-color: var(--el-fill-color-light); border-radius: 4px; padding: 12px; max-height: 150px; overflow-y: auto; border-left: 3px solid var(--el-color-primary);">
                      {{ currentEpisode.state_snapshot?.plot_summary || currentEpisode.description }}
                    </div>
                  </el-collapse-item>

                  <!-- Micro-beats (Agent 2 Builder) -->
                  <el-collapse-item v-if="currentEpisode.state_snapshot?.timeline && currentEpisode.description" title="Micro-beats (Agent 2 Builder)" name="beats">
                    <div style="white-space: pre-wrap; font-size: 13px; color: var(--text-secondary); line-height: 1.6; background-color: var(--el-fill-color-light); border-radius: 4px; padding: 12px; max-height: 150px; overflow-y: auto; border-left: 3px solid var(--accent);">
                      {{ currentEpisode.description }}
                    </div>
                  </el-collapse-item>

                  <!-- State Snapshot (Agent 2 State Tracking) -->
                  <el-collapse-item v-slot="{ active }" v-if="currentEpisode.state_snapshot && currentEpisode.state_snapshot.timeline" title="State Snapshot (Agent 2 State Tracking)" name="state">
                    <div style="font-size: 13px; background-color: var(--el-fill-color-light); border-radius: 4px; padding: 12px; max-height: 150px; overflow-y: auto; border-left: 3px solid var(--warning);">
                      <div v-if="currentEpisode.state_snapshot.timeline">
                        <strong>Timeline:</strong> {{ currentEpisode.state_snapshot.timeline }}
                      </div>
                      <div v-if="currentEpisode.state_snapshot.character_statuses" style="margin-top: 8px;">
                        <strong>Characters:</strong> {{ currentEpisode.state_snapshot.character_statuses }}
                      </div>
                      <div v-if="currentEpisode.state_snapshot.key_items_locations" style="margin-top: 8px;">
                        <strong>Key Items:</strong> {{ currentEpisode.state_snapshot.key_items_locations }}
                      </div>
                    </div>
                  </el-collapse-item>
                </el-collapse>
              </div>

              <el-divider />

              <!-- 显示已提取的角色和场景 -->
              <div v-if="hasExtractedData" class="extracted-info">
                <el-alert
                  type="success"
                  :closable="false"
                  style="margin-bottom: 16px"
                >
                  <template #title>
                    <div style="display: flex; align-items: center; gap: 16px">
                      <span>✅ {{ $t("workflow.extractedData") }}</span>
                      <el-tag v-if="hasCharacters" type="success"
                        >{{ $t("workflow.characters") }}:
                        {{ charactersCount }}</el-tag
                      >
                      <el-tag v-if="currentEpisode?.scenes" type="success"
                        >{{ $t("workflow.scenes") }}:
                        {{ currentEpisode.scenes.length }}</el-tag
                      >
                      <el-tag v-if="episodeOutfits.length > 0" type="success"
                        >Outfits:
                        {{ episodeOutfits.length }}</el-tag
                      >
                    </div>
                  </template>
                </el-alert>

                <!-- 角色列表 (Text List) -->
                <div v-if="currentEpisode?.characters && currentEpisode.characters.length > 0" style="margin-bottom: 24px">
                  <h4 class="extracted-title" style="font-size: 14px; font-weight: 600; color: var(--text-secondary); margin-bottom: 12px;">
                    {{ $t("workflow.extractedCharacters") }}：
                  </h4>
                  <div style="display: flex; flex-wrap: wrap; gap: 10px;">
                    <el-tag
                      v-for="char in currentEpisode.characters"
                      :key="char.id"
                      class="char-text-tag"
                      effect="plain"
                      type="info"
                      style="font-size: 13px; padding: 6px 12px; height: auto; border-radius: 6px;"
                    >
                      <strong style="color: var(--text-primary);">{{ char.name }}</strong>
                      <span style="margin: 0 6px; color: var(--border-primary);">|</span>
                      <span :style="{ color: char.role === 'main' ? 'var(--el-color-danger)' : 'var(--text-secondary)' }" style="font-weight: 600;">
                        {{ char.role === "main" ? "Main" : char.role === "supporting" ? "Supporting" : "Minor" }}
                      </span>
                      <span style="margin: 0 6px; color: var(--border-primary);">|</span>
                      <el-tag size="small" :type="hasImage(char) ? 'success' : 'warning'" effect="dark" style="border: none; border-radius: 4px; font-weight: 600;">
                        {{ hasImage(char) ? 'Library' : 'NEW' }}
                      </el-tag>
                    </el-tag>
                  </div>
                </div>

                <!-- 场景列表 (Text List) -->
                <div v-if="currentEpisode?.scenes && currentEpisode.scenes.length > 0" style="margin-bottom: 24px">
                  <h4 class="extracted-title" style="font-size: 14px; font-weight: 600; color: var(--text-secondary); margin-bottom: 12px;">
                    {{ $t("workflow.extractedScenes") }}：
                  </h4>
                  <div style="display: flex; flex-wrap: wrap; gap: 10px;">
                    <el-tag
                      v-for="scene in currentEpisode.scenes"
                      :key="scene.id"
                      class="scene-text-tag"
                      effect="plain"
                      type="info"
                      style="font-size: 13px; padding: 6px 12px; height: auto; border-radius: 6px;"
                    >
                      <strong style="color: var(--text-primary);">{{ scene.location }}</strong>
                      <span style="margin: 0 6px; color: var(--border-primary);">|</span>
                      <span style="color: var(--accent); font-weight: 600;">{{ scene.time }}</span>
                      <span style="margin: 0 6px; color: var(--border-primary);">|</span>
                      <el-tag size="small" :type="hasImage(scene) ? 'primary' : 'warning'" effect="dark" style="border: none; border-radius: 4px; font-weight: 600;">
                        {{ hasImage(scene) ? 'Reused' : 'NEW' }}
                      </el-tag>
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-card>

        <!-- 阶段 1: 生成图片 -->
        <el-card v-show="currentStep === 1" class="workflow-card">
          <div class="stage-body">
            <!-- 角色图片生成 -->
            <div class="image-gen-section">
              <div class="section-header">
                <div class="section-title">
                  <h3>
                    <el-icon><User /></el-icon>
                    {{ $t("workflow.characterImages") }}
                  </h3>
                  <el-alert type="info" :closable="false" style="margin: 0">
                    {{
                      $t("workflow.characterCount", { count: charactersCount })
                    }}
                  </el-alert>
                </div>
                <div class="section-actions">
                  <el-checkbox
                    v-model="selectAllCharacters"
                    @change="toggleSelectAllCharacters"
                    style="margin-right: 12px"
                  >
                    {{ $t("workflow.selectAll") }}
                  </el-checkbox>
                  <el-button
                    type="primary"
                    @click="batchGenerateOutfitImages"
                    :loading="batchGeneratingOutfits"
                    :disabled="selectedOutfitIds.length === 0"
                    size="default"
                    style="margin-left: 12px"
                  >
                    Generate Outfits ({{ selectedOutfitIds.length }})
                  </el-button>
                  <el-button
                    type="primary"
                    @click="batchGenerateCharacterImages"
                    :loading="batchGeneratingCharacters"
                    :disabled="selectedCharacterIds.length === 0"
                    size="default"
                  >
                    {{ $t("workflow.batchGenerate") }} ({{
                      selectedCharacterIds.length
                    }})
                  </el-button>
                </div>
              </div>

              <div class="character-image-list">
                <div
                  v-for="char in currentEpisode?.characters"
                  :key="char.id"
                  class="character-group"
                >
                    <!-- Character Main Card -->
                    <div class="character-item">
                      <el-card shadow="hover" class="character-card portrait-optimized-card" :body-style="{ padding: '0px' }">
                        <div class="card-image-container" :style="{ aspectRatio: cardAspectRatio }">
                          <el-image
                            v-if="hasImage(char)"
                            :src="getImageUrl(char)"
                            fit="cover"
                            :preview-src-list="[getImageUrl(char)]"
                            hide-on-click-modal
                            preview-teleported
                            class="portrait-image"
                          />
                          <div
                            v-else-if="
                              char.image_generation_status === 'pending' ||
                              char.image_generation_status === 'processing' ||
                              generatingCharacterImages[char.id]
                            "
                            class="image-placeholder generating"
                          >
                            <el-icon :size="48" class="rotating"><Loading /></el-icon>
                            <span>{{ $t("common.generating") }}</span>
                          </div>
                          <div v-else class="image-placeholder">
                            <el-icon :size="48"><User /></el-icon>
                            <span>{{ $t("common.notGenerated") }}</span>
                          </div>

                          <div class="card-overlay-premium">
                            <div class="overlay-top" style="display: flex; justify-content: space-between; width: 100%; align-items: center; pointer-events: auto;">
                              <div style="display: flex; align-items: center; gap: 8px;">
                                <el-checkbox
                                  v-model="selectedCharacterIds"
                                  :value="char.id"
                                />
                                <el-tag v-if="char.role" size="small" effect="dark" class="role-tag">
                                  {{ char.role === "main" ? "Main" : char.role === "supporting" ? "Supporting" : "Minor" }}
                                </el-tag>
                              </div>
                              <el-button
                                type="danger"
                                size="small"
                                :icon="Delete"
                                circle
                                plain
                                @click="deleteCharacter(char.id)"
                                class="delete-btn"
                                style="background: rgba(0,0,0,0.6); border: none; color: #ff4d4f;"
                              />
                            </div>
                            <div class="overlay-bottom">
                              <h4 class="char-name">{{ char.name }}</h4>
                            </div>
                          </div>
                        </div>

                        <div class="character-content" style="padding: 12px; background: var(--bg-card);">
                          <div class="card-actions-premium" style="display: flex; justify-content: center;">
                            <el-button-group>
                              <el-tooltip :content="$t('tooltip.editPrompt')" placement="top">
                                <el-button size="small" @click="openPromptDialog(char, 'character')" :icon="Edit" />
                              </el-tooltip>
                              <el-tooltip :content="$t('tooltip.aiGenerate')" placement="top">
                                <el-button type="primary" size="small" @click="generateCharacterImage(char.id)" :loading="generatingCharacterImages[char.id]" :icon="MagicStick" />
                              </el-tooltip>
                              <el-tooltip :content="$t('tooltip.uploadImage')" placement="top">
                                <el-button size="small" @click="uploadCharacterImage(char.id)" :icon="Upload" />
                              </el-tooltip>
                            </el-button-group>
                          </div>
                        </div>
                      </el-card>
                    </div>

                    <!-- Character Outfits Sub-Grid -->
                    <div v-if="char.outfits" class="outfits-sub-section">
                      <div class="outfit-section-header">
                        <span class="header-label">
                          <el-icon><Box /></el-icon>
                          Outfits ({{ char.outfits.length }})
                        </span>
                        <el-button size="small" type="primary" link :icon="Plus" @click="handleCreateOutfit(char)">Add Outfit</el-button>
                      </div>
                      <div class="outfit-grid">
                        <div v-for="outfit in char.outfits" :key="outfit.id" class="outfit-item">
                          <el-card shadow="hover" class="outfit-card-mini-workflow">
                            <div class="outfit-image-mini">
                              <el-image 
                                v-if="outfit.image_url || outfit.local_path"
                                :src="getImageUrl(outfit)" 
                                fit="cover" 
                                :preview-src-list="[getImageUrl(outfit)]"
                                preview-teleported
                              />
                              <div v-else class="outfit-placeholder-mini">
                                <el-icon v-if="!generatingOutfitImages[outfit.id]"><Box /></el-icon>
                                <el-icon v-else class="rotating"><Loading /></el-icon>
                              </div>
                              <div class="outfit-name-overlay">{{ outfit.name }}</div>
                            </div>
                            <div class="outfit-mini-actions">
                              <el-checkbox v-model="selectedOutfitIds" :value="outfit.id" size="small" />
                              <div class="action-btns">
                                <el-button type="primary" size="small" :icon="MagicStick" link @click="generateOutfitImage({ ...outfit, character_id: char.id })" :loading="generatingOutfitImages[outfit.id]" />
                                <el-button size="small" @click="uploadCharacterImage(outfit.id, 'outfit', char.id)" :icon="Upload" link />
                              </div>
                            </div>
                          </el-card>
                        </div>
                      </div>
                  </div>
                </div>
              </div>
            </div>

            <el-divider />

              <!-- 场景图片生成 -->
              <div class="image-gen-section">
              <div class="section-header">
                <div class="section-title">
                  <h3>
                    <el-icon><Place /></el-icon>
                    {{ $t("workflow.sceneImages") }}
                  </h3>
                  <el-alert type="info" :closable="false" style="margin: 0">
                    {{
                      $t("workflow.sceneCount", {
                        count: currentEpisode?.scenes?.length || 0,
                      })
                    }}
                  </el-alert>
                </div>
                <div class="section-actions">
                  <!-- <el-button
                  :icon="Document"
                  @click="openExtractSceneDialog"
                  size="default"
                >
                  {{ $t("workflow.extractFromScript") }}
                </el-button> -->
                  <el-checkbox
                    v-model="selectAllScenes"
                    @change="toggleSelectAllScenes"
                    style="margin-left: 12px; margin-right: 12px"
                  >
                    {{ $t("workflow.selectAll") }}
                  </el-checkbox>
                  <el-button
                    type="primary"
                    @click="batchGenerateSceneImages"
                    :loading="batchGeneratingScenes"
                    :disabled="selectedSceneIds.length === 0"
                    size="default"
                  >
                    {{ $t("workflow.batchGenerateSelected") }} ({{
                      selectedSceneIds.length
                    }})
                  </el-button>

                  <el-button
                    :icon="Plus"
                    @click="openAddSceneDialog"
                    size="default"
                  >
                    {{ $t("workflow.addScene") }}
                  </el-button>
                </div>
              </div>

              <div class="scene-image-list">
                <div
                  v-for="scene in currentEpisode?.scenes"
                  :key="scene.id"
                  class="scene-item"
                >
                  <el-card shadow="hover" class="scene-card portrait-optimized-card" :body-style="{ padding: '0px' }">
                    <div class="card-image-container" :style="{ aspectRatio: cardAspectRatio }">
                      <el-image
                        v-if="hasImage(scene)"
                        :src="getImageUrl(scene)"
                        fit="cover"
                        :preview-src-list="[getImageUrl(scene)]"
                        hide-on-click-modal
                        preview-teleported
                        class="portrait-image"
                      />
                      <div
                        v-else-if="
                          scene.image_generation_status === 'pending' ||
                          scene.image_generation_status === 'processing' ||
                          generatingSceneImages[scene.id]
                        "
                        class="image-placeholder generating"
                      >
                        <el-icon :size="48" class="rotating"><Loading /></el-icon>
                        <span>{{ $t("common.generating") }}</span>
                      </div>
                      <div v-else class="image-placeholder">
                        <el-icon :size="48"><Location /></el-icon>
                        <span>{{ $t("common.notGenerated") }}</span>
                      </div>

                      <div class="card-overlay-premium">
                        <div class="overlay-top" style="display: flex; justify-content: space-between; width: 100%; align-items: center; pointer-events: auto;">
                          <div style="display: flex; align-items: center; gap: 8px;">
                            <el-checkbox
                              v-model="selectedSceneIds"
                              :value="scene.id"
                            />
                            <div style="color: white; font-size: 20px; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5));">
                              <el-icon><LocationInformation /></el-icon>
                            </div>
                          </div>
                          <el-tag size="small" class="role-tag time-tag" effect="dark" style="background: rgba(0,0,0,0.6); border: none; color: #409eff; display: flex; align-items: center; gap: 4px;">
                            <el-icon><Clock /></el-icon> {{ scene.time }}
                          </el-tag>
                        </div>
                        <div class="overlay-bottom">
                          <h4 class="scene-location">{{ scene.location }}</h4>
                        </div>
                      </div>
                    </div>

                    <div class="scene-actions" style="padding: 12px; background: var(--bg-card); display: flex; justify-content: center; gap: 8px;">
                      <el-button size="small" @click="openPromptDialog(scene, 'scene')">{{ $t("common.edit") }}</el-button>
                      <el-button size="small" type="primary" @click="generateSceneImage(scene.id)" :loading="generatingSceneImages[scene.id]">
                        <el-icon style="margin-right: 4px"><MagicStick /></el-icon> {{ $t("prop.generateImage") }}
                      </el-button>
                      <el-button size="small" @click="uploadSceneImage(scene.id)" :icon="Upload" />
                    </div>
                  </el-card>
                </div>
              </div>
            </div>

          </div>
        </el-card>

        <!-- 阶段 2: 拆分分镜 -->
        <el-card v-show="currentStep === 2" shadow="never" class="stage-card">
          <div class="stage-body">
            <!-- 分镜列表 -->
            <div
              v-if="
                currentEpisode?.storyboards &&
                currentEpisode.storyboards.length > 0
              "
              class="shots-list"
            >
              <div class="shots-header">
                <h3>{{ $t("workflow.shotList") }}</h3>
                <div class="shots-batch-actions">
                  <el-checkbox
                    v-model="selectAllStoryboards"
                    @change="toggleSelectAllStoryboards"
                  >
                    {{ $t("workflow.selectAll") }}
                  </el-checkbox>
                  <el-button
                    type="primary"
                    :loading="batchGeneratingFramePrompts"
                    :disabled="selectedStoryboardIds.length === 0"
                    @click="batchGenerateFirstFramePrompts"
                  >
                    {{ $t("workflow.batchGenerateFirstFrame") }}
                    <span v-if="selectedStoryboardIds.length > 0">
                      ({{ selectedStoryboardIds.length }})
                    </span>
                  </el-button>
                  <el-button
                    type="warning"
                    :loading="batchGeneratingLtxVideoPrompts"
                    :disabled="selectedStoryboardIds.length === 0"
                    @click="batchGenerateLtxVideoPrompts"
                  >
                    {{ $t("workflow.batchGenerateLtxVideoPrompt") }}
                    <span v-if="selectedStoryboardIds.length > 0">
                      ({{ selectedStoryboardIds.length }})
                    </span>
                  </el-button>
                  <el-button
                    type="danger"
                    :loading="batchGeneratingVideos"
                    :disabled="selectedStoryboardIds.length === 0"
                    @click="batchGenerateVideos"
                  >
                    {{ $t("workflow.batchGenerateVideo") }}
                    <span v-if="selectedStoryboardIds.length > 0">
                      ({{ selectedStoryboardIds.length }})
                    </span>
                  </el-button>
                  <el-button
                    type="success"
                    :loading="batchGeneratingStoryboardImages"
                    :disabled="selectedStoryboardIds.length === 0"
                    @click="batchGenerateStoryboardImages"
                  >
                    {{ $t("workflow.batchGenerateShotImage") }}
                    <span v-if="selectedStoryboardIds.length > 0">
                      ({{ selectedStoryboardIds.length }})
                    </span>
                  </el-button>
                  <el-button
                    :icon="Film"
                    :loading="exportingFullVideo"
                    @click="exportFullEpisodeVideo"
                  >
                    {{ $t("workflow.exportFullVideo") }}
                  </el-button>
                  <el-button
                    :icon="Picture"
                    :loading="exportingShotImages"
                    @click="exportShotImagesZip"
                  >
                    {{ $t("workflow.exportShotImages") }}
                  </el-button>
                </div>
              </div>

              <el-table
                ref="storyboardTableRef"
                :data="currentEpisode.storyboards"
                border
                stripe
                style="margin-top: 16px"
                @selection-change="handleStoryboardSelectionChange"
              >
                <el-table-column type="selection" width="50" />
                <el-table-column
                  type="index"
                  :label="$t('storyboard.table.number')"
                  width="60"
                />
                <el-table-column
                  :label="$t('storyboard.table.title')"
                  width="120"
                  show-overflow-tooltip
                >
                  <template #default="{ row }">
                    {{ row.title || "-" }}
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.shotType')"
                  width="80"
                >
                  <template #default="{ row }">
                    {{ row.shot_type || "-" }}
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.movement')"
                  width="80"
                >
                  <template #default="{ row }">
                    {{ row.movement || "-" }}
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.location')"
                  width="150"
                >
                  <template #default="{ row }">
                    <el-popover
                      placement="right"
                      :width="300"
                      trigger="hover"
                      :content="row.action || '-'"
                    >
                      <template #reference>
                        <!-- 单行打点 -->
                        <span class="overflow-tooltip">{{
                          row.location || "-"
                        }}</span>
                      </template>
                    </el-popover>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.character')"
                  width="150"
                >
                  <template #default="{ row }">
                    <div v-if="row.characters && row.characters.length > 0">
                      <div v-for="char in row.characters" :key="char.id" style="margin-bottom: 2px">
                        <el-tag size="small" effect="plain">{{ char.name }}</el-tag>
                        <span v-if="getShotOutfitName(row, char.id)" class="shot-outfit-text">
                          : {{ getShotOutfitName(row, char.id) }}
                        </span>
                      </div>
                    </div>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('storyboard.table.action')">
                  <template #default="{ row }">
                    <el-popover
                      placement="right"
                      :width="300"
                      trigger="hover"
                      :content="row.action || '-'"
                    >
                      <template #reference>
                        <!-- 单行打点 -->
                        <span class="overflow-tooltip">{{
                          row.action || "-"
                        }}</span>
                      </template>
                    </el-popover>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.duration')"
                  width="80"
                >
                  <template #default="{ row }">
                    {{ row.duration || "-" }}s
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('workflow.firstFrameStatus')"
                  width="100"
                >
                  <template #default="{ row }">
                    <el-tag
                      v-if="hasFirstFramePrompt(row.id)"
                      type="success"
                      size="small"
                    >
                      {{ $t("workflow.firstFramePrompted") }}
                    </el-tag>
                    <el-tag v-else type="info" size="small" effect="plain">
                      -
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('workflow.ltxVideoPromptStatus')"
                  width="120"
                >
                  <template #default="{ row }">
                    <el-tag
                      v-if="row.ltx_video_prompt"
                      type="success"
                      size="small"
                    >
                      {{ $t("workflow.ltxVideoPromptReady") }}
                    </el-tag>
                    <el-tag
                      v-else-if="ltxVideoPromptGeneratingShots[Number(row.id)]"
                      type="warning"
                      size="small"
                    >
                      {{ $t("workflow.ltxVideoPromptGenerating") }}
                    </el-tag>
                    <el-tag v-else type="info" size="small" effect="plain">
                      -
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('workflow.videoGenStatus')"
                  width="120"
                >
                  <template #default="{ row }">
                    <el-tag
                      v-if="videoBatchSubmittingShots[Number(row.id)]"
                      type="warning"
                      size="small"
                    >
                      {{ $t("workflow.videoGenSubmitting") }}
                    </el-tag>
                    <el-tag
                      v-else-if="
                        latestVideoByStoryboard[Number(row.id)]?.status ===
                        'completed'
                      "
                      type="success"
                      size="small"
                    >
                      {{ $t("workflow.videoGenReady") }}
                    </el-tag>
                    <el-tag
                      v-else-if="
                        latestVideoByStoryboard[Number(row.id)]?.status ===
                          'pending' ||
                        latestVideoByStoryboard[Number(row.id)]?.status ===
                          'processing'
                      "
                      type="warning"
                      size="small"
                    >
                      {{ $t("workflow.videoGenProcessing") }}
                    </el-tag>
                    <el-tag
                      v-else-if="
                        latestVideoByStoryboard[Number(row.id)]?.status ===
                        'failed'
                      "
                      type="danger"
                      size="small"
                    >
                      {{ $t("workflow.videoGenFailed") }}
                    </el-tag>
                    <el-tag v-else type="info" size="small" effect="plain">
                      -
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('workflow.shotImageStatus')"
                  width="110"
                >
                  <template #default="{ row }">
                    <el-tag
                      v-if="row.composed_image"
                      type="success"
                      size="small"
                    >
                      {{ $t("workflow.shotImageReady") }}
                    </el-tag>
                    <el-tag
                      v-else-if="
                        row.image_generation_status === 'pending' ||
                        row.image_generation_status === 'processing'
                      "
                      type="warning"
                      size="small"
                    >
                      {{ $t("workflow.shotImageGenerating") }}
                    </el-tag>
                    <el-tag
                      v-else-if="row.image_generation_status === 'failed'"
                      type="danger"
                      size="small"
                    >
                      {{ $t("workflow.shotImageFailed") }}
                    </el-tag>
                    <el-tag v-else type="info" size="small" effect="plain">
                      {{ $t("workflow.shotImageNone") }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column
                  :label="$t('storyboard.table.operations')"
                  width="100"
                  fixed="right"
                >
                  <template #default="{ row, $index }">
                    <el-button
                      type="primary"
                      size="small"
                      @click="editShot(row, $index)"
                    >
                      {{ $t("common.edit") }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- 未拆分时显示 -->
            <div v-else class="empty-shots">
              <el-empty :description="$t('workflow.splitStoryboardFirst')">
                <el-button
                  type="primary"
                  @click="generateShots"
                  :loading="generatingShots"
                  :icon="MagicStick"
                >
                  {{
                    generatingShots
                      ? $t("workflow.aiSplitting")
                      : $t("workflow.aiAutoSplit")
                  }}
                </el-button>

                <!-- 任务进度显示 -->
                <div
                  v-if="generatingShots"
                  style="
                    margin-top: 24px;
                    max-width: 400px;
                    margin-left: auto;
                    margin-right: auto;
                  "
                >
                  <el-progress
                    :percentage="taskProgress"
                    :status="taskProgress === 100 ? 'success' : undefined"
                  >
                    <template #default="{ percentage }">
                      <span style="font-size: 12px">{{ percentage }}%</span>
                    </template>
                  </el-progress>
                  <div class="task-message">
                    {{ taskMessage }}
                  </div>
                </div>
              </el-empty>
            </div>
          </div>
        </el-card>
      </div>

      <div class="actions-container">
        <div class="action-buttons" v-show="currentStep === 0">
          <el-button
            type="primary"
            size="large"
            @click="handleExtractCharactersAndBackgrounds"
            :loading="extractingCharactersAndBackgrounds"
            :disabled="!hasScript"
          >
            <el-icon><MagicStick /></el-icon>
            {{
              hasExtractedData
                ? $t("workflow.reExtract")
                : "✨ Extract & Sync Entities"
            }}
          </el-button>
          <el-button
            type="success"
            size="large"
            @click="nextStep"
          >
            {{ $t("workflow.nextStepGenerateImages") }}
            <el-icon><ArrowRight /></el-icon>
          </el-button>
          <div v-if="!hasExtractedData" style="margin-top: 8px">
            <el-alert
              type="info"
              :closable="false"
              style="display: inline-block"
            >
              <template #title>
                <span style="font-size: 12px">
                  {{ $t("workflow.extractWarning") }}
                </span>
              </template>
            </el-alert>
          </div>
        </div>

        <div class="action-buttons" v-show="currentStep === 1">
          <el-button size="large" @click="prevStep">
            <el-icon><ArrowLeft /></el-icon>
            {{ $t("workflow.prevStep") }}
          </el-button>
          <el-button
            type="success"
            size="large"
            @click="nextStep"
          >
            {{ $t("workflow.nextStepSplitShots") }}
            <el-icon><ArrowRight /></el-icon>
          </el-button>
          <div v-if="!allImagesGenerated" style="margin-top: 8px">
            <el-alert
              type="info"
              :closable="false"
              style="display: inline-block"
            >
              <template #title>
                <span style="font-size: 12px">
                  {{ $t("workflow.generateAllImagesFirst") }}
                </span>
              </template>
            </el-alert>
          </div>
        </div>

        <div class="action-buttons" v-show="currentStep === 2">
          <el-button size="large" @click="prevStep">
            <el-icon><ArrowLeft /></el-icon>
            {{ $t("workflow.prevStep") }}
          </el-button>
          <el-button size="large" @click="regenerateShots" :icon="MagicStick">
            {{ $t("workflow.reSplitShots") }}
          </el-button>
          <el-button type="success" size="large" @click="goToProfessionalUI">
            {{ $t("workflow.enterProfessional") }}
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <div class="components-box">
      <!-- 镜头编辑对话框 -->
      <el-dialog
        v-model="shotEditDialogVisible"
        :title="$t('workflow.editShot')"
        width="800px"
        :close-on-click-modal="false"
      >
        <el-form v-if="editingShot" label-width="100px" size="default">
          <el-form-item :label="$t('workflow.shotTitle')">
            <el-input
              v-model="editingShot.title"
              :placeholder="$t('workflow.shotTitlePlaceholder')"
            />
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item :label="$t('workflow.shotType')">
                <el-select
                  v-model="editingShot.shot_type"
                  :placeholder="$t('workflow.selectShotType')"
                >
                  <el-option :label="$t('workflow.longShot')" value="Long Shot" />
                  <el-option :label="$t('workflow.fullShot')" value="Full Shot" />
                  <el-option :label="$t('workflow.mediumShot')" value="Medium Shot" />
                  <el-option :label="$t('workflow.closeUp')" value="Close Up" />
                  <el-option
                    :label="$t('workflow.extremeCloseUp')"
                    value="Extreme Close Up"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item :label="$t('workflow.cameraAngle')">
                <el-select
                  v-model="editingShot.angle"
                  :placeholder="$t('workflow.selectAngle')"
                >
                  <el-option :label="$t('workflow.eyeLevel')" value="Eye Level" />
                  <el-option :label="$t('workflow.lowAngle')" value="Low Angle" />
                  <el-option :label="$t('workflow.highAngle')" value="High Angle" />
                  <el-option :label="$t('workflow.sideView')" value="Side View" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item :label="$t('workflow.cameraMovement')">
                <el-select
                  v-model="editingShot.movement"
                  :placeholder="$t('workflow.selectMovement')"
                >
                  <el-option
                    :label="$t('workflow.staticShot')"
                    value="Static Shot"
                  />
                  <el-option :label="$t('workflow.pushIn')" value="Push In" />
                  <el-option :label="$t('workflow.pullOut')" value="Pull Out" />
                  <el-option :label="$t('workflow.followShot')" value="Follow Shot" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item :label="$t('workflow.location')">
                <el-input
                  v-model="editingShot.location"
                  :placeholder="$t('workflow.locationPlaceholder')"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="$t('workflow.time')">
                <el-input
                  v-model="editingShot.time"
                  :placeholder="$t('workflow.timeSetting')"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item :label="$t('workflow.shotDescription')">
            <el-input
              v-model="editingShot.description"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.shotDescriptionPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.actionDescription')">
            <el-input
              v-model="editingShot.action"
              type="textarea"
              :rows="3"
              :placeholder="$t('workflow.detailedAction')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.dialogue')">
            <el-input
              v-model="editingShot.dialogue"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.characterDialogue')"
            />
          </el-form-item>

          <el-form-item label="Narration">
            <el-input
              v-model="editingShot.narration"
              type="textarea"
              :rows="2"
              placeholder="Voiceover narration"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.result')">
            <el-input
              v-model="editingShot.result"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.actionResult')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.atmosphere')">
            <el-input
              v-model="editingShot.atmosphere"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.atmosphereDescription')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.imagePrompt')">
            <el-input
              v-model="editingShot.image_prompt"
              type="textarea"
              :rows="3"
              :placeholder="$t('workflow.imagePromptPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.videoPrompt')">
            <el-input
              v-model="editingShot.video_prompt"
              type="textarea"
              :rows="3"
              :placeholder="$t('workflow.videoPromptPlaceholder')"
            />
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item :label="$t('workflow.bgmHint')">
                <el-input
                  v-model="editingShot.bgm_prompt"
                  :placeholder="$t('workflow.bgmAtmosphere')"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="$t('workflow.soundEffect')">
                <el-input
                  v-model="editingShot.sound_effect"
                  :placeholder="$t('workflow.soundEffectDescription')"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item :label="$t('workflow.durationSeconds')">
            <el-input-number
              v-model="editingShot.duration"
              :min="1"
              :max="60"
            />
          </el-form-item>

          <el-divider content-position="left">Characters & Outfits</el-divider>
          
          <el-form-item label="Involved Characters">
            <el-checkbox-group v-model="editingShot.character_ids">
              <el-checkbox v-for="char in currentEpisode?.characters" :key="char.id" :label="char.id">
                {{ char.name }}
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>

          <div v-if="editingShot.character_ids && editingShot.character_ids.length > 0" class="outfit-assignments">
            <div v-for="charId in editingShot.character_ids" :key="charId" class="outfit-assign-row">
              <span class="char-name-label">{{ getCharacterName(charId) }}:</span>
              <el-select v-model="editingShot.character_outfits[charId]" placeholder="Default Outfit" clearable style="width: 200px">
                <el-option
                  v-for="outfit in getCharacterOutfits(charId)"
                  :key="outfit.id"
                  :label="outfit.name"
                  :value="outfit.id"
                />
              </el-select>
            </div>
          </div>
        </el-form>

        <template #footer>
          <el-button @click="shotEditDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            type="primary"
            @click="saveShotEdit"
            :loading="savingShot"
            >{{ $t("common.save") }}</el-button
          >
        </template>
      </el-dialog>

      <!-- 提示词编辑对话框 -->
      <el-dialog
        v-model="promptDialogVisible"
        :title="$t('workflow.editPrompt')"
        width="600px"
      >
        <el-form label-width="80px">
          <el-form-item :label="$t('common.name')">
            <el-input v-model="currentEditItem.name" disabled />
          </el-form-item>
          <el-form-item
            v-if="currentEditType === 'scene'"
            :label="$t('workflow.time')"
          >
            <el-input
              v-model="currentEditItem.time"
              :placeholder="$t('workflow.timePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.imagePrompt')">
            <el-input
              v-model="editPrompt"
              type="textarea"
              :rows="6"
              :placeholder="$t('workflow.imagePromptPlaceholder')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="promptDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="savePrompt">{{
            $t("common.save")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 角色库选择对话框 -->
      <el-dialog
        v-model="libraryDialogVisible"
        :title="$t('workflow.selectFromLibrary')"
        width="800px"
      >
        <div class="library-grid">
          <div
            v-for="item in libraryItems"
            :key="item.id"
            class="library-item"
            @click="selectLibraryItem(item)"
          >
            <el-image :src="getImageUrl(item)" fit="cover" />
            <div class="library-item-name">{{ item.name }}</div>
          </div>
        </div>
        <div v-if="libraryItems.length === 0" class="empty-library">
          <el-empty :description="$t('workflow.emptyLibrary')" />
        </div>
      </el-dialog>

      <!-- AI模型配置对话框 -->
      <el-dialog
        v-model="modelConfigDialogVisible"
        :title="$t('workflow.aiModelConfig')"
        width="600px"
        :close-on-click-modal="false"
      >
        <el-form label-width="120px">
          <el-form-item :label="$t('workflow.textGenModel')">
            <el-select
              v-model="selectedTextModel"
              :placeholder="$t('workflow.selectTextModel')"
              style="width: 100%"
            >
              <el-option
                v-for="model in textModels"
                :key="model.modelName"
                :label="model.modelName"
                :value="model.modelName"
              />
            </el-select>
            <div class="model-tip">
              {{ $t("workflow.textModelTip") }}
            </div>
          </el-form-item>

          <el-form-item :label="$t('workflow.imageGenModel')">
            <el-select
              v-model="selectedImageModel"
              :placeholder="$t('workflow.selectImageModel')"
              style="width: 100%"
            >
              <el-option
                v-for="model in imageModels"
                :key="model.modelName"
                :label="model.modelName"
                :value="model.modelName"
              />
            </el-select>
            <div class="model-tip">
              {{ $t("workflow.modelConfigTip") }}
            </div>
          </el-form-item>

          <el-form-item :label="$t('workflow.videoGenModel')">
            <el-select
              v-model="selectedVideoModel"
              :placeholder="$t('workflow.selectVideoModel')"
              style="width: 100%"
              clearable
            >
              <el-option
                v-for="model in videoModels"
                :key="model.modelName"
                :label="model.modelName"
                :value="model.modelName"
              />
            </el-select>
            <div class="model-tip">
              {{ $t("workflow.videoModelTip") }}
            </div>
          </el-form-item>
        </el-form>

        <template #footer>
          <el-button @click="modelConfigDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveModelConfig">{{
            $t("common.saveConfig")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 图片上传对话框 -->
      <el-dialog
        v-model="uploadDialogVisible"
        :title="$t('tooltip.uploadImage')"
        width="500px"
      >
        <el-upload
          class="upload-area"
          drag
          :action="uploadAction"
          :headers="uploadHeaders"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :show-file-list="false"
          accept="image/jpeg,image/png,image/jpg"
        >
          <el-icon class="el-icon--upload"><Upload /></el-icon>
          <div class="el-upload__text">
            {{ $t("workflow.dragFilesHere")
            }}<em>{{ $t("workflow.clickToUpload") }}</em>
          </div>
          <template #tip>
            <div class="el-upload__tip">
              {{ $t("workflow.uploadFormatTip") }}
            </div>
          </template>
        </el-upload>
      </el-dialog>

      <!-- 添加场景对话框 -->
      <el-dialog
        v-model="addSceneDialogVisible"
        :title="$t('workflow.addScene')"
        width="600px"
      >
        <el-form :model="newScene" label-width="100px">
          <el-form-item :label="$t('workflow.sceneImage')">
            <el-upload
              class="avatar-uploader"
              :action="`/api/v1/upload/image`"
              :show-file-list="false"
              :on-success="handleSceneImageSuccess"
              :before-upload="beforeAvatarUpload"
            >
              <img
                v-if="hasImage(newScene)"
                :src="getImageUrl(newScene)"
                class="avatar"
                style="width: 160px; height: 90px; object-fit: cover"
              />
              <el-icon
                v-else
                class="avatar-uploader-icon"
                style="
                  border: 1px dashed #d9d9d9;
                  border-radius: 6px;
                  cursor: pointer;
                  position: relative;
                  overflow: hidden;
                  width: 160px;
                  height: 90px;
                  font-size: 28px;
                  color: #8c939d;
                  text-align: center;
                  line-height: 90px;
                "
                ><Plus
              /></el-icon>
            </el-upload>
          </el-form-item>
          <el-form-item :label="$t('workflow.sceneName')">
            <el-input
              v-model="newScene.location"
              :placeholder="$t('workflow.sceneNamePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.time')">
            <el-input
              v-model="newScene.time"
              :placeholder="$t('workflow.timePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.sceneDescription')">
            <el-input
              v-model="newScene.prompt"
              type="textarea"
              :rows="4"
              :placeholder="$t('workflow.sceneDescriptionPlaceholder')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="addSceneDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveScene">{{
            $t("common.confirm")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 从剧本提取场景对话框 -->
      <el-dialog
        v-model="extractScenesDialogVisible"
        :title="$t('workflow.extractSceneDialogTitle')"
        width="500px"
      >
        <el-alert type="info" :closable="false" style="margin-bottom: 16px">
          {{ $t("workflow.extractSceneDialogTip") }}
        </el-alert>
        <template #footer>
          <el-button @click="extractScenesDialogVisible = false">
            {{ $t("common.cancel") }}
          </el-button>
          <el-button
            type="primary"
            @click="handleExtractScenes"
            :loading="extractingScenes"
          >
            {{ $t("workflow.startExtract") }}
          </el-button>
        </template>
      </el-dialog>

      <!-- 添加服装对话框 -->
      <el-dialog
        v-model="outfitDialogVisible"
        title="Add New Outfit"
        width="500px"
      >
        <el-form :model="outfitForm" label-width="100px">
          <el-form-item label="Outfit Name">
            <el-input v-model="outfitForm.name" placeholder="e.g. Prison Uniform, Wedding Dress" />
          </el-form-item>
          <el-form-item label="Description">
            <el-input
              v-model="outfitForm.prompt"
              type="textarea"
              :rows="4"
              placeholder="Detailed description of the outfit for AI generation"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="outfitDialogVisible = false">{{ $t("common.cancel") }}</el-button>
          <el-button type="primary" @click="saveOutfit" :loading="savingOutfit">{{ $t("common.confirm") }}</el-button>
        </template>
      </el-dialog>

      <!-- Edit Graph Connections Dialog -->
      <el-dialog
        v-model="graphDialogVisible"
        title="Edit Narrative Connections"
        width="680px"
      >
        <div style="margin-bottom: 20px; padding: 12px 16px; background: rgba(99,102,241,0.08); border-radius: 8px; border-left: 3px solid #6366f1;">
          <p style="margin: 0; font-size: 13px; color: var(--text-secondary); line-height: 1.6;">
            Editing connections for <strong style="color: var(--el-color-primary);">Episode {{ episodeNumber }} ({{ currentEpisode?.narrative_node_id || 'N/A' }})</strong>.
            You can configure both <em>which episodes lead here</em> (Incoming) and <em>which choices branch off from here</em> (Outgoing).
          </p>
        </div>

        <!-- SECTION 1: Incoming connections (which parent episodes point TO this one) -->
        <div style="margin-bottom: 20px;">
          <div style="font-size: 12px; font-weight: 700; color: #94a3b8; letter-spacing: 0.8px; text-transform: uppercase; margin-bottom: 10px; display: flex; align-items: center; gap: 8px;">
            <span style="background: rgba(100,116,139,0.2); border: 1px solid rgba(100,116,139,0.4); border-radius: 4px; padding: 2px 8px;">← INCOMING</span>
            <span style="font-weight: 400; color: var(--text-muted);">Episodes that have a choice pointing to this episode</span>
          </div>
          <div style="display: flex; flex-direction: column; gap: 8px;">
            <div
              v-for="(inc, idx) in editableIncoming"
              :key="'inc-'+idx"
              style="display: flex; align-items: center; gap: 12px; background: rgba(100,116,139,0.08); padding: 10px 12px; border-radius: 6px; border: 1px solid rgba(100,116,139,0.2);"
            >
              <div style="flex: 1;">
                <div style="font-size: 11px; font-weight: 700; color: #94a3b8; margin-bottom: 4px;">SOURCE EPISODE</div>
                <el-select v-model="inc.source_episode_id" placeholder="Select Parent Episode" clearable style="width: 100%;">
                  <el-option
                    v-for="ep in otherEpisodes"
                    :key="ep.id"
                    :label="`Ep ${ep.episode_number}: ${ep.title} (${ep.narrative_node_id || 'N/A'})`"
                    :value="ep.id"
                  />
                </el-select>
              </div>
              <div style="flex: 1;">
                <div style="font-size: 11px; font-weight: 700; color: #94a3b8; margin-bottom: 4px;">CHOICE LABEL ON THAT EPISODE</div>
                <el-input v-model="inc.label" placeholder="e.g. Follow the mysterious stranger" />
              </div>
              <div style="align-self: flex-end; padding-bottom: 2px;">
                <el-button type="danger" :icon="Delete" circle @click="removeIncoming(idx)" />
              </div>
            </div>
            <div v-if="editableIncoming.length === 0" style="text-align: center; padding: 16px; background: rgba(100,116,139,0.05); border-radius: 6px; border: 1px dashed rgba(100,116,139,0.2);">
              <span style="font-size: 12px; color: var(--text-muted);">No incoming connections. Episode is isolated or an entry point.</span>
            </div>
          </div>
          <el-button
            size="small"
            @click="addIncoming"
            style="margin-top: 8px; background: rgba(100,116,139,0.1); border: 1px solid rgba(100,116,139,0.3); color: #94a3b8;"
          >
            + Add Incoming Connection
          </el-button>
        </div>

        <el-divider style="margin: 16px 0;" />

        <!-- SECTION 2: Outgoing choices (what branches off from here) -->
        <div>
          <div style="font-size: 12px; font-weight: 700; color: #a5b4fc; letter-spacing: 0.8px; text-transform: uppercase; margin-bottom: 10px; display: flex; align-items: center; gap: 8px;">
            <span style="background: rgba(99,102,241,0.15); border: 1px solid rgba(99,102,241,0.4); border-radius: 4px; padding: 2px 8px;">→ OUTGOING</span>
            <span style="font-weight: 400; color: var(--text-muted);">Choices that branch off from this episode</span>
          </div>

          <div v-if="editableChoices.length === 0" style="text-align: center; padding: 24px; background: rgba(99,102,241,0.05); border-radius: 6px; border: 1px dashed rgba(99,102,241,0.2); margin-bottom: 12px;">
            <span style="font-size: 12px; color: var(--text-muted);">No outgoing choices. This is a terminal episode.</span>
          </div>

          <div v-else style="display: flex; flex-direction: column; gap: 10px; margin-bottom: 12px; max-height: 280px; overflow-y: auto; padding-right: 4px;">
            <div
              v-for="(choice, idx) in editableChoices"
              :key="idx"
              style="display: flex; align-items: center; gap: 12px; background: rgba(99,102,241,0.08); padding: 10px 12px; border-radius: 6px; border: 1px solid rgba(99,102,241,0.2);"
            >
              <div style="flex: 1;">
                <div style="font-size: 11px; font-weight: 700; color: #a5b4fc; margin-bottom: 4px;">CHOICE LABEL</div>
                <el-input v-model="choice.label" placeholder="e.g. Accept Elena's offer" />
              </div>
              <div style="flex: 1;">
                <div style="font-size: 11px; font-weight: 700; color: #a5b4fc; margin-bottom: 4px;">TARGET EPISODE</div>
                <el-select v-model="choice.next_episode_id" placeholder="Select Target" clearable style="width: 100%;">
                  <el-option
                    v-for="ep in otherEpisodes"
                    :key="ep.id"
                    :label="`Ep ${ep.episode_number}: ${ep.title}`"
                    :value="ep.id"
                  />
                </el-select>
              </div>
              <div style="align-self: flex-end; padding-bottom: 2px;">
                <el-button type="danger" :icon="Delete" circle @click="removeGraphChoice(idx)" />
              </div>
            </div>
          </div>

          <el-button
            size="small"
            @click="addGraphChoice"
            style="width: 100%; background: rgba(99,102,241,0.1); border: 1px solid rgba(99,102,241,0.35); color: #a5b4fc;"
          >
            + Add Outgoing Branch / Choice
          </el-button>
        </div>

        <template #footer>
          <el-button @click="graphDialogVisible = false">{{ $t("common.cancel") }}</el-button>
          <el-button type="primary" @click="saveGraphChoices">Save Connections</el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  User,
  Location,
  Picture,
  MagicStick,
  ArrowRight,
  ArrowLeft,
  Place,
  Film,
  Edit,
  More,
  Upload,
  Delete,
  FolderAdd,
  Setting,
  Loading,
  WarningFilled,
  Document,
  Plus,
} from "@element-plus/icons-vue";
import { dramaAPI } from "@/api/drama";
import { generationAPI } from "@/api/generation";
import { characterLibraryAPI } from "@/api/character-library";
import { aiAPI } from "@/api/ai";
import type { AIServiceConfig } from "@/types/ai";
import { imageAPI } from "@/api/image";
import { ltxVideoPromptAPI } from "@/api/ltx-video-prompt";
import { videoAPI } from "@/api/video";
import type { ImageGeneration } from "@/types/image";
import type { VideoGeneration } from "@/types/video";
import type { Drama } from "@/types/drama";
import { buildShotImagesZip } from "@/utils/exportShotImagesZip";
import { videoMerger } from "@/utils/videoMerger";
import { AppHeader } from "@/components/common";
import { getImageUrl, getVideoUrl, hasImage } from "@/utils/image";
import {
  generateFirstFrame,
  getEpisodeFramePrompts,
  getStoryboardFramePrompts,
} from "@/api/frame";
import type { FramePromptRecord } from "@/api/frame";
import { taskAPI } from "@/api/task";

const route = useRoute();
const router = useRouter();
const { t: $t } = useI18n();
const dramaId = route.params.id as string;
const episodeNumber = parseInt(route.params.episodeNumber as string);

const drama = ref<Drama>();

// 生成 localStorage key
const getStepStorageKey = () =>
  `episode_workflow_step_${dramaId}_${episodeNumber}`;

// 从 localStorage 恢复步骤，如果没有则默认为 0
const savedStep = localStorage.getItem(getStepStorageKey());
const isPortrait = computed(() => drama.value?.aspect_ratio === '9:16');
const cardAspectRatio = computed(() => isPortrait.value ? '9/16' : '3/4');

const currentStep = ref(Math.max(0, Math.min(2, savedStep ? parseInt(savedStep) : 0)));
const scriptContent = ref("");
const isEditingScript = ref(false);
const generatingScript = ref(false);
const generatingShots = ref(false);
const extractingCharactersAndBackgrounds = ref(false);
const batchGeneratingCharacters = ref(false);
const batchGeneratingScenes = ref(false);
const generatingCharacterImages = ref<Record<number, boolean>>({});
const generatingSceneImages = ref<Record<string, boolean>>({});
const generatingOutfitImages = ref<Record<number, boolean>>({});

// 选择状态
const selectedCharacterIds = ref<number[]>([]);
const selectedSceneIds = ref<number[]>([]);
const selectedOutfitIds = ref<number[]>([]);
const selectAllCharacters = ref(false);
const selectAllScenes = ref(false);
const selectAllOutfits = ref(false);
const batchGeneratingOutfits = ref(false);

// Storyboard batch first-frame prompt state
const selectedStoryboardIds = ref<number[]>([]);
const selectAllStoryboards = ref(false);
const batchGeneratingFramePrompts = ref(false);
const batchGeneratingStoryboardImages = ref(false);
const batchGeneratingLtxVideoPrompts = ref(false);
const ltxVideoPromptGeneratingShots = ref<Record<number, boolean>>({});
const batchGeneratingVideos = ref(false);
const videoBatchSubmittingShots = ref<Record<number, boolean>>({});
const latestVideoByStoryboard = ref<Record<number, VideoGeneration>>({});
const exportingShotImages = ref(false);
const exportingFullVideo = ref(false);
const episodeFramePrompts = ref<Record<string, FramePromptRecord[]>>({});

// 对话框状态
const promptDialogVisible = ref(false);
const libraryDialogVisible = ref(false);
const uploadDialogVisible = ref(false);
const modelConfigDialogVisible = ref(false);
const addSceneDialogVisible = ref(false);
const extractScenesDialogVisible = ref(false);
const currentEditItem = ref<any>({ name: "" });
const currentEditType = ref<"character" | "scene">("character");
const editPrompt = ref("");
const libraryItems = ref<any[]>([]);
const currentUploadTarget = ref<any>(null);

// Outfit 相关
const outfitDialogVisible = ref(false);
const graphDialogVisible = ref(false);
const editableChoices = ref<any[]>([]);
const editableIncoming = ref<any[]>([]); // NEW: incoming connections from parent episodes
const outfitForm = ref({
  name: "",
  prompt: "",
  character_id: 0,
});
const savingOutfit = ref(false);

// 添加场景相关
const newScene = ref<any>({
  location: "",
  time: "",
  prompt: "",
  image_url: "",
  local_path: "",
});
const extractingScenes = ref(false);
const uploadAction = computed(() => "/api/v1/upload/image");
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem("token")}`,
}));

// AI模型配置
interface ModelOption {
  modelName: string;
  configName: string;
  configId: number;
  priority: number;
}

const textModels = ref<ModelOption[]>([]);
const imageModels = ref<ModelOption[]>([]);
const videoModels = ref<ModelOption[]>([]);
const selectedTextModel = ref<string>("");
const selectedImageModel = ref<string>("");
const selectedVideoModel = ref<string>("");

const hasScript = computed(() => {
  const currentEp = currentEpisode.value;
  return (
    currentEp && currentEp.script_content && currentEp.script_content.length > 0
  );
});

const currentEpisode = computed(() => {
  if (!drama.value?.episodes) return null;
  return drama.value.episodes.find((ep) => ep.episode_number === episodeNumber);
});

const hasCharacters = computed(() => {
  return (
    currentEpisode.value?.characters &&
    currentEpisode.value.characters.length > 0
  );
});

const charactersCount = computed(() => {
  return currentEpisode.value?.characters?.length || 0;
});

const episodeOutfits = computed(() => {
  const outfits: any[] = [];
  currentEpisode.value?.characters?.forEach((char: any) => {
    char.outfits?.forEach((o: any) => {
      outfits.push({
        ...o,
        character_name: char.name,
        character_id: char.id,
      });
    });
  });
  return outfits;
});

// Graph Path Indicators (Previous & Next Episodes)
const previousEpisodes = computed(() => {
  if (!drama.value?.episodes || !currentEpisode.value) return [];
  const currNodeId = currentEpisode.value.narrative_node_id;
  const currEpId = String(currentEpisode.value.id);
  const currEpNum = currentEpisode.value.episode_number;
  
  const parents = drama.value.episodes.filter((ep) => {
    if (String(ep.id) === currEpId) return false;
    return ep.choices?.some((choice) => {
      return (
        (currNodeId && choice.next_narrative_node_id === currNodeId) ||
        String(choice.next_episode_id) === currEpId
      );
    });
  });

  if (parents.length === 0 && currEpNum > 1) {
    const prev = drama.value.episodes.find((ep) => ep.episode_number === currEpNum - 1);
    if (prev) parents.push(prev);
  }
  
  return parents;
});

const nextEpisodes = computed(() => {
  if (!drama.value?.episodes || !currentEpisode.value) return [];
  const children: any[] = [];
  
  if (currentEpisode.value.choices?.length) {
    currentEpisode.value.choices.forEach((choice) => {
      const child = drama.value!.episodes?.find((ep) => {
        return (
          (choice.next_narrative_node_id && ep.narrative_node_id === choice.next_narrative_node_id) ||
          (choice.next_episode_id && String(ep.id) === String(choice.next_episode_id))
        );
      });
      if (child) children.push(child);
    });
  }

  if (children.length === 0) {
    const next = drama.value.episodes.find((ep) => ep.episode_number === currentEpisode.value!.episode_number + 1);
    if (next) children.push(next);
  }

  return children;
});

const goToEpisode = (num: number) => {
  // Direct redirect is the safest way to clear and reload the production workflow state cleanly
  window.location.href = `/dramas/${dramaId}/episodes/${num}`;
};

const otherEpisodes = computed(() => {
  if (!drama.value?.episodes) return [];
  return drama.value.episodes.filter(ep => ep.episode_number !== episodeNumber);
});

const openEditGraphDialog = () => {
  editableChoices.value = currentEpisode.value?.choices?.map((c: any) => ({
    label: c.label,
    next_episode_id: c.next_episode_id,
    next_narrative_node_id: c.next_narrative_node_id,
  })) || [];

  // Build editable incoming list: find all parent episodes that have a choice pointing here
  const currNodeId = currentEpisode.value?.narrative_node_id;
  const currEpId = String(currentEpisode.value?.id);
  editableIncoming.value = [];
  drama.value?.episodes?.forEach((ep) => {
    if (String(ep.id) === currEpId) return;
    ep.choices?.forEach((c: any) => {
      if (
        (currNodeId && c.next_narrative_node_id === currNodeId) ||
        String(c.next_episode_id) === currEpId
      ) {
        editableIncoming.value.push({
          source_episode_id: ep.id,
          label: c.label || '',
        });
      }
    });
  });

  graphDialogVisible.value = true;
};

const addIncoming = () => {
  editableIncoming.value.push({ source_episode_id: undefined, label: '' });
};

const removeIncoming = (index: number) => {
  editableIncoming.value.splice(index, 1);
};

const addGraphChoice = () => {
  editableChoices.value.push({
    label: `Option ${editableChoices.value.length + 1}`,
    next_episode_id: undefined,
  });
};

const removeGraphChoice = (index: number) => {
  editableChoices.value.splice(index, 1);
};

const saveGraphChoices = async () => {
  if (!drama.value?.episodes || !currentEpisode.value) return;

  try {
    const existingEpisodes = [...drama.value.episodes];
    const currEpId = String(currentEpisode.value.id);
    const currNodeId = currentEpisode.value.narrative_node_id;

    // 1. Update OUTGOING choices of the current episode
    const episodeIndex = existingEpisodes.findIndex(
      (ep) => ep.episode_number === episodeNumber,
    );
    if (episodeIndex >= 0) {
      existingEpisodes[episodeIndex] = {
        ...existingEpisodes[episodeIndex],
        choices: editableChoices.value
          .filter((c: any) => c.next_episode_id)
          .map((c: any) => {
            const targetEp = drama.value?.episodes?.find(e => String(e.id) === String(c.next_episode_id));
            return {
              label: c.label || 'Continue',
              next_episode_id: Number(c.next_episode_id),
              next_narrative_node_id: targetEp?.narrative_node_id || undefined,
            };
          }),
      };
    }

    // 2. Update INCOMING: reconcile parent episodes' choices that point to current episode
    // First strip all existing choices pointing here from any parent
    existingEpisodes.forEach((ep, idx) => {
      if (String(ep.id) === currEpId) return;
      const filtered = (ep.choices || []).filter((c: any) =>
        c.next_narrative_node_id !== currNodeId &&
        String(c.next_episode_id) !== currEpId
      );
      existingEpisodes[idx] = { ...ep, choices: filtered };
    });

    // Then add back the configured incoming connections
    editableIncoming.value
      .filter((inc: any) => inc.source_episode_id)
      .forEach((inc: any) => {
        const parentIdx = existingEpisodes.findIndex(e => String(e.id) === String(inc.source_episode_id));
        if (parentIdx < 0) return;
        const parentChoices = [...(existingEpisodes[parentIdx].choices || [])];
        parentChoices.push({
          label: inc.label || 'Continue',
          next_episode_id: Number(currEpId),
          next_narrative_node_id: currNodeId || undefined,
        });
        existingEpisodes[parentIdx] = { ...existingEpisodes[parentIdx], choices: parentChoices };
      });

    await dramaAPI.saveEpisodes(dramaId, existingEpisodes);
    ElMessage.success('Graph connections saved successfully');
    graphDialogVisible.value = false;
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to update graph connections');
  }
};

const hasExtractedData = computed(() => {
  const hasScenes =
    currentEpisode.value?.scenes && currentEpisode.value.scenes.length > 0;
  // 只要有角色或场景，就认为已经提取过数据
  return hasCharacters.value || hasScenes;
});

const allImagesGenerated = computed(() => {
  // 如果没有提取任何数据，允许跳过（可能是空章节或用户想直接进入拆解分镜）
  if (!hasExtractedData.value) return true;

  const characters = currentEpisode.value?.characters || [];
  const scenes = currentEpisode.value?.scenes || [];

  // 如果角色和场景都为空，允许跳过
  if (characters.length === 0 && scenes.length === 0) return true;

  // 检查所有有数据的项是否都已生成图片
  const allCharsHaveImages =
    characters.length === 0 || characters.every((char) => char.image_url);
  const allScenesHaveImages =
    scenes.length === 0 || scenes.every((scene) => scene.image_url);

  return allCharsHaveImages && allScenesHaveImages;
});

const goBack = () => {
  // 使用 replace 避免在历史记录中留下当前页面
  router.replace(`/dramas/${dramaId}`);
};

// 加载AI模型配置
const loadAIConfigs = async () => {
  try {
    const [textList, imageList, videoList] = await Promise.all([
      aiAPI.list("text"),
      aiAPI.list("image"),
      aiAPI.list("video"),
    ]);

    // 只使用激活的配置
    const activeTextList = textList.filter((c) => c.is_active);
    const activeImageList = imageList.filter((c) => c.is_active);
    const activeVideoList = videoList.filter((c) => c.is_active);

    // 展开模型列表并去重（保留优先级最高的）
    const allTextModels = activeTextList
      .flatMap((config) => {
        const models = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return models.map((modelName) => ({
          modelName,
          configName: config.name,
          configId: config.id,
          priority: config.priority || 0,
        }));
      })
      .sort((a, b) => b.priority - a.priority);

    // 按模型名称去重，保留优先级最高的（已排序，第一个就是优先级最高的）
    const textModelMap = new Map<string, ModelOption>();
    allTextModels.forEach((model) => {
      if (!textModelMap.has(model.modelName)) {
        textModelMap.set(model.modelName, model);
      }
    });
    textModels.value = Array.from(textModelMap.values());

    const allImageModels = activeImageList
      .flatMap((config) => {
        const models = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return models.map((modelName) => ({
          modelName,
          configName: config.name,
          configId: config.id,
          priority: config.priority || 0,
        }));
      })
      .sort((a, b) => b.priority - a.priority);

    // 按模型名称去重，保留优先级最高的
    const imageModelMap = new Map<string, ModelOption>();
    allImageModels.forEach((model) => {
      if (!imageModelMap.has(model.modelName)) {
        imageModelMap.set(model.modelName, model);
      }
    });
    imageModels.value = Array.from(imageModelMap.values());

    const allVideoModels = activeVideoList
      .flatMap((config) => {
        const models = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return models.map((modelName) => ({
          modelName,
          configName: config.name,
          configId: config.id,
          priority: config.priority || 0,
        }));
      })
      .sort((a, b) => b.priority - a.priority);

    const videoModelMap = new Map<string, ModelOption>();
    allVideoModels.forEach((model) => {
      if (!videoModelMap.has(model.modelName)) {
        videoModelMap.set(model.modelName, model);
      }
    });
    videoModels.value = Array.from(videoModelMap.values());

    // 设置默认选择（优先级最高的）
    if (textModels.value.length > 0 && !selectedTextModel.value) {
      selectedTextModel.value = textModels.value[0].modelName;
    }
    if (imageModels.value.length > 0 && !selectedImageModel.value) {
      // 优先选择包含 nano 的模型
      const nanoModel = imageModels.value.find((m) =>
        m.modelName.toLowerCase().includes("nano"),
      );
      selectedImageModel.value = nanoModel
        ? nanoModel.modelName
        : imageModels.value[0].modelName;
    }
    if (videoModels.value.length > 0 && !selectedVideoModel.value) {
      selectedVideoModel.value = videoModels.value[0].modelName;
    }

    // 验证已选择的模型是否还在可用列表中，如果不在则重置为默认值
    const availableTextModelNames = textModels.value.map((m) => m.modelName);
    const availableImageModelNames = imageModels.value.map((m) => m.modelName);
    const availableVideoModelNames = videoModels.value.map((m) => m.modelName);

    if (
      selectedTextModel.value &&
      !availableTextModelNames.includes(selectedTextModel.value)
    ) {
      console.warn(
        `Selected text model ${selectedTextModel.value} is unavailable, reset to default`,
      );
      selectedTextModel.value =
        textModels.value.length > 0 ? textModels.value[0].modelName : "";
      // 更新 localStorage
      if (selectedTextModel.value) {
        localStorage.setItem(
          `ai_text_model_${dramaId}`,
          selectedTextModel.value,
        );
      }
    }

    if (
      selectedImageModel.value &&
      !availableImageModelNames.includes(selectedImageModel.value)
    ) {
      console.warn(
        `Selected image model ${selectedImageModel.value} is unavailable, reset to default`,
      );
      // 优先选择包含 nano 的模型
      const nanoModel = imageModels.value.find((m) =>
        m.modelName.toLowerCase().includes("nano"),
      );
      selectedImageModel.value =
        imageModels.value.length > 0
          ? nanoModel
            ? nanoModel.modelName
            : imageModels.value[0].modelName
          : "";
      // 更新 localStorage
      if (selectedImageModel.value) {
        localStorage.setItem(
          `ai_image_model_${dramaId}`,
          selectedImageModel.value,
        );
      }
    }

    if (
      selectedVideoModel.value &&
      availableVideoModelNames.length > 0 &&
      !availableVideoModelNames.includes(selectedVideoModel.value)
    ) {
      selectedVideoModel.value = videoModels.value[0]?.modelName || "";
      if (selectedVideoModel.value) {
        localStorage.setItem(
          `ai_video_model_${dramaId}`,
          selectedVideoModel.value,
        );
      }
    }
  } catch (error: any) {
    console.error("Failed to load AI configs:", error);
  }
};

// 显示模型配置对话框
const showModelConfigDialog = () => {
  modelConfigDialogVisible.value = true;
  loadAIConfigs();
};

// 保存模型配置
const saveModelConfig = () => {
  if (!selectedTextModel.value || !selectedImageModel.value) {
    ElMessage.warning($t("workflow.pleaseSelectModels"));
    return;
  }

  // 保存模型名称到localStorage
  localStorage.setItem(`ai_text_model_${dramaId}`, selectedTextModel.value);
  localStorage.setItem(`ai_image_model_${dramaId}`, selectedImageModel.value);
  if (selectedVideoModel.value) {
    localStorage.setItem(`ai_video_model_${dramaId}`, selectedVideoModel.value);
  } else {
    localStorage.removeItem(`ai_video_model_${dramaId}`);
  }

  ElMessage.success($t("workflow.modelConfigSaved"));
  modelConfigDialogVisible.value = false;
};

const nextStep = () => {
  if (currentStep.value < 2) {
    currentStep.value++;
  }
};

const prevStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--;
  }
};

const goToStep = (step: number) => {
  currentStep.value = Math.max(0, Math.min(2, step));
};

// 从localStorage加载已保存的模型配置
const loadSavedModelConfig = () => {
  const savedTextModel = localStorage.getItem(`ai_text_model_${dramaId}`);
  const savedImageModel = localStorage.getItem(`ai_image_model_${dramaId}`);
  const savedVideoModel = localStorage.getItem(`ai_video_model_${dramaId}`);

  if (savedTextModel) {
    selectedTextModel.value = savedTextModel;
  }
  if (savedImageModel) {
    selectedImageModel.value = savedImageModel;
  }
  if (savedVideoModel) {
    selectedVideoModel.value = savedVideoModel;
  }
};

const loadDramaData = async () => {
  try {
    const data = await dramaAPI.get(dramaId);
    drama.value = data;

    if (!hasScript.value && currentStep.value === 0) {
      scriptContent.value = "";
      // 如果没有剧本内容，重置到第一步
    }

    // Enrich storyboard characters via the dedicated storyboards endpoint,
    // which reliably returns characters (GET /dramas/:id nested preload is unreliable for many2many)
    await enrichStoryboardCharacters();

    // 检查是否有生成中的角色或场景，自动启动轮询
    await checkAndStartPolling();

    // Load frame prompts now that episode data is available
    await loadEpisodeFramePrompts();

    await loadEpisodeVideos();
  } catch (error: any) {
    ElMessage.error(error.message || "Failed to load project data");
  }
};

const enrichStoryboardCharacters = async () => {
  const ep = currentEpisode.value;
  if (!ep?.id || !ep.storyboards?.length) return;
  try {
    const sbData: any = await dramaAPI.getStoryboards(String(ep.id));
    const enrichedMap = new Map<number, any>(
      (sbData.storyboards || []).map((sb: any) => [sb.id, sb]),
    );
    const epIdx = drama.value?.episodes?.findIndex(
      (e: any) => e.episode_number === episodeNumber,
    );
    if (epIdx !== undefined && epIdx >= 0 && drama.value?.episodes) {
      drama.value.episodes[epIdx].storyboards = drama.value.episodes[
        epIdx
      ].storyboards!.map((sb: any) => {
        const en =
          enrichedMap.get(sb.id) ?? enrichedMap.get(Number(sb.id));
        return {
          ...sb,
          characters: en?.characters ?? sb.characters ?? [],
          background: en?.background ?? sb.background,
          ltx_video_prompt: en?.ltx_video_prompt ?? sb.ltx_video_prompt,
          composed_image: en?.composed_image ?? sb.composed_image,
          image_generation_id: en?.image_generation_id ?? sb.image_generation_id,
          image_generation_status:
            en?.image_generation_status ?? sb.image_generation_status,
        };
      });
    }
  } catch (e) {
    console.warn("Failed to enrich storyboard characters:", e);
  }
};

// 检查并启动轮询
const checkAndStartPolling = async () => {
  if (!currentEpisode.value) return;

  // 检查角色的生成状态
  for (const char of currentEpisode.value.characters || []) {
    if (
      char.image_generation_status === "pending" ||
      char.image_generation_status === "processing"
    ) {
      // 查找对应的image_generation记录
      try {
        const imageGenList = await imageAPI.listImages({
          drama_id: dramaId,
          status: char.image_generation_status as any,
        });

        // 找到这个角色的image_generation记录
        const charImageGen = imageGenList.items.find(
          (img) =>
            img.character_id === char.id &&
            (img.status === "pending" || img.status === "processing"),
        );

        if (charImageGen) {
          // 启动轮询
          generatingCharacterImages.value[char.id] = true;
          pollImageStatus(charImageGen.id, async () => {
            await loadDramaData();
            ElMessage.success(`${char.name} image generation completed`);
          }).finally(() => {
            generatingCharacterImages.value[char.id] = false;
          });
        }
      } catch (error) {
        console.error("[Poll] Failed to query character image generation:", error);
      }
    }
  }

  // 检查场景的生成状态
  for (const scene of currentEpisode.value.scenes || []) {
    if (
      scene.image_generation_status === "pending" ||
      scene.image_generation_status === "processing"
    ) {
      // 查找对应的image_generation记录
      try {
        const imageGenList = await imageAPI.listImages({
          drama_id: dramaId,
          status: scene.image_generation_status as any,
        });

        // 找到这个场景的image_generation记录
        const sceneImageGen = imageGenList.items.find(
          (img) =>
            img.scene_id === scene.id &&
            (img.status === "pending" || img.status === "processing"),
        );

        if (sceneImageGen) {
          // 启动轮询
          generatingSceneImages.value[scene.id] = true;
          pollImageStatus(sceneImageGen.id, async () => {
            await loadDramaData();
            ElMessage.success(`${scene.location} image generation completed`);
          }).finally(() => {
            generatingSceneImages.value[scene.id] = false;
          });
        }
      } catch (error) {
        console.error("[Poll] Failed to query scene image generation:", error);
      }
    }
  }
};

const saveChapterScript = async () => {
  try {
    const existingEpisodes = drama.value?.episodes || [];

    // 查找当前章节
    const episodeIndex = existingEpisodes.findIndex(
      (ep) => ep.episode_number === episodeNumber,
    );

    let updatedEpisodes;
    if (episodeIndex >= 0) {
      // 更新已有章节
      updatedEpisodes = [...existingEpisodes];
      updatedEpisodes[episodeIndex] = {
        ...updatedEpisodes[episodeIndex],
        script_content: scriptContent.value,
      };
    } else {
      // 创建新章节
      const newEpisode = {
        episode_number: episodeNumber,
        title: `Episode ${episodeNumber}`,
        script_content: scriptContent.value,
      };
      updatedEpisodes = [...existingEpisodes, newEpisode];
    }

    await dramaAPI.saveEpisodes(dramaId, updatedEpisodes);
    ElMessage.success("Episode saved");
    isEditingScript.value = false;
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "Save failed");
  }
};

const editCurrentEpisodeScript = () => {
  scriptContent.value = currentEpisode.value?.script_content || "";
  isEditingScript.value = true;
};

const handleExtractCharactersAndBackgrounds = async () => {
  // 如果已经提取过，显示确认对话框
  if (hasExtractedData.value) {
    try {
      await ElMessageBox.confirm(
        $t("workflow.reExtractConfirmMessage"),
        $t("workflow.reExtractConfirmTitle"),
        {
          confirmButtonText: $t("common.confirm"),
          cancelButtonText: $t("common.cancel"),
          type: "warning",
          distinguishCancelAndClose: true,
        },
      );
    } catch {
      ElMessage.info($t("workflow.extractCancelled"));
      return;
    }
  }

  // 显示即将开始的提示
  if (hasExtractedData.value) {
    ElMessage.info($t("workflow.startReExtracting"));
  }

  await extractCharactersAndBackgrounds();
};

// 轮询检查图片生成状态
const pollImageStatus = async (
  imageGenId: number,
  onComplete: () => Promise<void>,
) => {
  const maxAttempts = 100; // 最多轮询100次
  const pollInterval = 6000; // 每6秒轮询一次

  for (let i = 0; i < maxAttempts; i++) {
    try {
      await new Promise((resolve) => setTimeout(resolve, pollInterval));

      const imageGen = await imageAPI.getImage(imageGenId);

      if (imageGen.status === "completed") {
        // 生成成功
        await onComplete();
        return;
      } else if (imageGen.status === "failed") {
        // 生成失败
        ElMessage.error(`Image generation failed: ${imageGen.error_msg || "Unknown error"}`);
        return;
      }
      // 如果是pending或processing，继续轮询
    } catch (error: any) {
      console.error("[Poll] Failed to check image status:", error);
      // 继续轮询，不中断
    }
  }

  // 超时
  ElMessage.warning("Image generation timed out. Please refresh later.");
};

const extractCharactersAndBackgrounds = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("Episode info not found");
    return;
  }

  extractingCharactersAndBackgrounds.value = true;

  try {
    const episodeId = currentEpisode.value.id;

    // 并行创建异步任务
    const [characterTask, backgroundTask] = await Promise.all([
      characterLibraryAPI.extractFromEpisode(episodeId),
      dramaAPI.extractBackgrounds(
        episodeId.toString(),
        selectedTextModel.value,
      ), // 传递用户选择的文本模型
    ]);

    ElMessage.success("Task created and processing in background...");

    // 并行轮询两个任务
    await Promise.all([
      pollExtractTask(characterTask.task_id, "character"),
      pollExtractTask(backgroundTask.task_id, "background"),
    ]);

    ElMessage.success($t("workflow.charactersAndScenesExtractSuccess"));
    await loadDramaData();
  } catch (error: any) {
    console.error($t("workflow.charactersAndScenesExtractFailed") + ":", error);

    const errorData = error.response?.data?.error;
    const errorMsg = errorData?.message || error.message || "Extraction failed";

    if (
      errorMsg.includes("no config found") ||
      errorMsg.includes("AI client") ||
      errorMsg.includes("failed to get AI client")
    ) {
      ElMessage({
        type: "warning",
        message: 'AI service is not configured. Go to "Settings > AI Config" and add a text service.',
        duration: 5000,
        showClose: true,
      });
    } else {
      ElMessage.error(errorMsg);
    }
  } finally {
    extractingCharactersAndBackgrounds.value = false;
  }
};

// 轮询提取任务状态
const pollExtractTask = async (
  taskId: string,
  type: "character" | "background",
) => {
  const maxAttempts = 60; // 最多轮询60次（2分钟）
  const interval = 2000; // 每2秒查询一次

  for (let i = 0; i < maxAttempts; i++) {
    await new Promise((resolve) => setTimeout(resolve, interval));

    try {
      const task = await generationAPI.getTaskStatus(taskId);

      if (task.status === "completed") {
        // 任务完成
        if (type === "character" && task.result) {
          // 解析角色数据并保存
          const result =
            typeof task.result === "string"
              ? JSON.parse(task.result)
              : task.result;
          if (result.characters && result.characters.length > 0) {
            await dramaAPI.saveCharacters(
              dramaId,
              result.characters,
              currentEpisode.value?.id,
            );
          }
        }
        return;
      } else if (task.status === "failed") {
        // 任务失败
        throw new Error(
          task.error ||
            (type === "character"
              ? $t("workflow.characterGenerationFailed")
              : $t("workflow.sceneExtractionFailed")),
        );
      }
      // 否则继续轮询
    } catch (error: any) {
      console.error(`Failed to poll ${type} task status:`, error);
      throw error;
    }
  }

  throw new Error(
    type === "character"
      ? $t("workflow.characterGenerationTimeout")
      : $t("workflow.sceneExtractionTimeout"),
  );
};

const generateCharacterImage = async (characterId: number) => {
  generatingCharacterImages.value[characterId] = true;

  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;
    const response = await characterLibraryAPI.generateCharacterImage(
      characterId.toString(),
      model,
    );
    const imageGenId = response.image_generation?.id;

    if (imageGenId) {
      ElMessage.info("Generating character image...");
      // 轮询检查生成状态
      await pollImageStatus(imageGenId, async () => {
        await loadDramaData();
        ElMessage.success("Character image generated");
      });
    } else {
      ElMessage.success("Character image generation started");
      await loadDramaData();
    }
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  } finally {
    generatingCharacterImages.value[characterId] = false;
  }
};

const toggleSelectAllCharacters = () => {
  if (selectAllCharacters.value) {
    selectedCharacterIds.value =
      currentEpisode.value?.characters?.map((char) => char.id) || [];
  } else {
    selectedCharacterIds.value = [];
  }
};

const toggleSelectAllScenes = () => {
  if (selectAllScenes.value) {
    selectedSceneIds.value =
      currentEpisode.value?.scenes?.map((scene) => scene.id) || [];
  } else {
    selectedSceneIds.value = [];
  }
};

const batchGenerateCharacterImages = async () => {
  if (selectedCharacterIds.value.length === 0) {
    ElMessage.warning("Please select characters first");
    return;
  }

  batchGeneratingCharacters.value = true;
  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;

    // 使用批量生成API
    await characterLibraryAPI.batchGenerateCharacterImages(
      selectedCharacterIds.value.map((id) => id.toString()),
      model,
    );

    ElMessage.success($t("workflow.batchTaskSubmitted"));
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchGenerateFailed"));
  } finally {
    batchGeneratingCharacters.value = false;
  }
};

const generateOutfitImage = async (outfit: any) => {
  generatingOutfitImages.value[outfit.id] = true;

  try {
    const model = selectedImageModel.value || undefined;
    const response = await characterLibraryAPI.generateOutfitImage(
      outfit.character_id,
      outfit.id,
      { model },
    );
    const imageGenId = response.image_generation?.id;

    if (imageGenId) {
      ElMessage.info("Generating outfit image...");
      await pollImageStatus(imageGenId, async () => {
        await loadDramaData();
        ElMessage.success("Outfit image generated");
      });
    } else {
      ElMessage.success("Outfit image generation started");
      await loadDramaData();
    }
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  } finally {
    generatingOutfitImages.value[outfit.id] = false;
  }
};

const toggleSelectAllOutfits = () => {
  if (selectAllOutfits.value) {
    selectedOutfitIds.value = episodeOutfits.value.map((o) => o.id);
  } else {
    selectedOutfitIds.value = [];
  }
};

const batchGenerateOutfitImages = async () => {
  if (selectedOutfitIds.value.length === 0) {
    ElMessage.warning("Please select outfits first");
    return;
  }

  batchGeneratingOutfits.value = true;
  try {
    const model = selectedImageModel.value || undefined;
    const outfitsToGen = episodeOutfits.value.filter((o) =>
      selectedOutfitIds.value.includes(o.id),
    );

    const promises = outfitsToGen.map((o) => generateOutfitImage(o));
    await Promise.allSettled(promises);

    ElMessage.success("Batch outfit generation complete");
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "Batch generation failed");
  } finally {
    batchGeneratingOutfits.value = false;
  }
};

const handleCreateOutfit = (char: any) => {
  outfitForm.value = {
    name: "",
    prompt: "",
    character_id: char.id,
  };
  outfitDialogVisible.value = true;
};

const saveOutfit = async () => {
  if (!outfitForm.value.name) {
    ElMessage.warning("Please enter outfit name");
    return;
  }

  savingOutfit.value = true;
  try {
    await characterLibraryAPI.createOutfit(outfitForm.value.character_id, {
      name: outfitForm.value.name,
      prompt: outfitForm.value.prompt,
    });
    ElMessage.success("Outfit added");
    outfitDialogVisible.value = false;
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "Failed to add outfit");
  } finally {
    savingOutfit.value = false;
  }
};

const generateSceneImage = async (sceneId: string) => {
  generatingSceneImages.value[sceneId] = true;

  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;
    const response = await dramaAPI.generateSceneImage({
      scene_id: parseInt(sceneId),
      model,
    });
    const imageGenId = response.image_generation?.id;

    if (imageGenId) {
      ElMessage.info($t("workflow.sceneImageGenerating"));
      // 轮询检查生成状态
      await pollImageStatus(imageGenId, async () => {
        await loadDramaData();
        ElMessage.success($t("workflow.sceneImageComplete"));
      });
    } else {
      ElMessage.success($t("workflow.sceneImageStarted"));
      await loadDramaData();
    }
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  } finally {
    generatingSceneImages.value[sceneId] = false;
  }
};

const batchGenerateSceneImages = async () => {
  if (selectedSceneIds.value.length === 0) {
    ElMessage.warning("Please select scenes first");
    return;
  }

  batchGeneratingScenes.value = true;
  try {
    const promises = selectedSceneIds.value.map((sceneId) =>
      generateSceneImage(sceneId.toString()),
    );
    const results = await Promise.allSettled(promises);

    const successCount = results.filter((r) => r.status === "fulfilled").length;
    const failCount = results.filter((r) => r.status === "rejected").length;

    if (failCount === 0) {
      ElMessage.success(
        $t("workflow.batchCompleteSuccess", { count: successCount }),
      );
    } else {
      ElMessage.warning(
        $t("workflow.batchCompletePartial", {
          success: successCount,
          fail: failCount,
        }),
      );
    }
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchGenerateFailed"));
  } finally {
    batchGeneratingScenes.value = false;
  }
};

const taskProgress = ref(0);
const taskMessage = ref("");
let pollTimer: any = null;

const generateShots = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("Episode info not found");
    return;
  }

  generatingShots.value = true;
  taskProgress.value = 0;
  taskMessage.value = "Initializing task...";

  try {
    const episodeId = currentEpisode.value.id.toString();

    // 【调试日志】输出当前操作的集数信息
    console.log("=== Start storyboard generation ===");
    console.log("Current episodeNumber (route param):", episodeNumber);
    console.log("Current episodeId (from currentEpisode):", episodeId);
    console.log("currentEpisode details:", {
      id: currentEpisode.value?.id,
      episode_number: currentEpisode.value?.episode_number,
      title: currentEpisode.value?.title,
    });
    console.log(
      "All episodes:",
      drama.value?.episodes?.map((ep) => ({
        id: ep.id,
        episode_number: ep.episode_number,
        title: ep.title,
      })),
    );

    // 创建异步任务
    const response = await generationAPI.generateStoryboard(
      episodeId,
      selectedTextModel.value,
    );

    taskMessage.value = response.message || "Task created";

    // 开始轮询任务状态
    await pollTaskStatus(response.task_id);
  } catch (error: any) {
    ElMessage.error(error.message || "Split failed");
    generatingShots.value = false;
  }
};

const pollTaskStatus = async (taskId: string) => {
  const checkStatus = async () => {
    try {
      const task = await generationAPI.getTaskStatus(taskId);

      taskProgress.value = task.progress;
      taskMessage.value = task.message || `Processing... ${task.progress}%`;

      if (task.status === "completed") {
        // 任务完成
        if (pollTimer) {
          clearInterval(pollTimer);
          pollTimer = null;
        }
        generatingShots.value = false;

        ElMessage.success($t("workflow.splitSuccess"));

        // 跳转到专业编辑器页面
        router.push({
          name: "ProfessionalEditor",
          params: {
            dramaId: dramaId,
            episodeNumber: episodeNumber,
          },
        });
      } else if (task.status === "failed") {
        // 任务失败
        if (pollTimer) {
          clearInterval(pollTimer);
          pollTimer = null;
        }
        generatingShots.value = false;
        ElMessage.error(task.error || "Storyboard split failed");
      }
      // 否则继续轮询
    } catch (error: any) {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
      generatingShots.value = false;
      ElMessage.error("Failed to query task status: " + error.message);
    }
  };

  // 立即检查一次
  await checkStatus();

  // 每2秒轮询一次
  pollTimer = setInterval(checkStatus, 2000);
};

const regenerateShots = async () => {
  await ElMessageBox.confirm($t("workflow.reSplitConfirm"), $t("common.tip"), {
    type: "warning",
  });

  await generateShots();
};

const shotEditDialogVisible = ref(false);
const editingShot = ref<any>(null);
const editingShotIndex = ref<number>(-1);
const savingShot = ref(false);

const editShot = (shot: any, index: number) => {
  // Prepare character ids and outfits
  const charIds = (shot.characters || []).map((c: any) => typeof c === 'object' ? c.id : c);
  const charOutfits: Record<number, number | null> = {};
  
  // If shot.characters is a list of objects, they might have pivot data
  // But usually we need to check how the backend returns it.
  // Assuming shot.characters has been preloaded with pivot data if possible,
  // or we just use the current mapping if we have it.
  (shot.characters || []).forEach((c: any) => {
    if (c.pivot?.outfit_id) {
      charOutfits[c.id] = c.pivot.outfit_id;
    } else if (c.outfit_id) {
      charOutfits[c.id] = c.outfit_id;
    }
  });

  editingShot.value = { 
    ...shot, 
    character_ids: charIds,
    character_outfits: charOutfits
  };
  editingShotIndex.value = index;
  shotEditDialogVisible.value = true;
};

const getCharacterName = (id: number) => {
  const char = currentEpisode.value?.characters?.find((c: any) => c.id === id);
  return char?.name || `Character ${id}`;
};

const getCharacterOutfits = (id: number) => {
  const char = currentEpisode.value?.characters?.find((c: any) => c.id === id);
  return char?.outfits || [];
};

const getShotOutfitName = (shot: any, charId: number) => {
  // Check if character has pivot data with outfit_id
  const char = shot.characters?.find((c: any) => c.id === charId);
  const outfitId = char?.pivot?.outfit_id || char?.outfit_id;
  if (!outfitId) return null;
  
  // Find outfit name in the global character library (loaded in currentEpisode)
  const fullChar = currentEpisode.value?.characters?.find((c: any) => c.id === charId);
  const outfit = fullChar?.outfits?.find((o: any) => o.id === outfitId);
  return outfit?.name || null;
};

const saveShotEdit = async () => {
  if (!editingShot.value) return;

  try {
    savingShot.value = true;

    // Prepare character data with outfits for backend
    const charactersPayload = (editingShot.value.character_ids || []).map((id: number) => ({
      id: id,
      outfit_id: editingShot.value.character_outfits[id] || null
    }));

    const updatePayload = {
      ...editingShot.value,
      characters: charactersPayload
    };

    // 调用API更新镜头
    await dramaAPI.updateStoryboard(
      editingShot.value.id.toString(),
      updatePayload,
    );

    // Update local data - reload drama to get latest associations
    await loadDramaData();

    ElMessage.success("Shot updated");
    shotEditDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error("Save failed: " + (error.message || "Unknown error"));
  } finally {
    savingShot.value = false;
  }
};

// 对话框相关方法
const openPromptDialog = (item: any, type: "character" | "scene") => {
  currentEditItem.value = item;
  currentEditItem.value.name = item.name || item.location;
  currentEditType.value = type;
  editPrompt.value = item.prompt || item.appearance || item.description || "";
  promptDialogVisible.value = true;
};

const savePrompt = async () => {
  try {
    if (currentEditType.value === "character") {
      await characterLibraryAPI.updateCharacter(currentEditItem.value.id, {
        appearance: editPrompt.value,
      });
      await generateCharacterImage(currentEditItem.value.id);
    } else {
      // 保存场景提示词和时间（合并到一个 API 调用）
      await dramaAPI.updateScene(currentEditItem.value.id.toString(), {
        prompt: editPrompt.value,
        time: currentEditItem.value.time || "",
      });

      ElMessage.success("Saved");
      await loadDramaData();
    }
    promptDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error(error.message || "Save failed");
  }
};

const uploadCharacterImage = (
  id: number | string,
  type: "character" | "outfit" = "character",
  parentId?: number,
) => {
  currentUploadTarget.value = { id, type, parentId };
  uploadDialogVisible.value = true;
};

const uploadSceneImage = (sceneId: string) => {
  currentUploadTarget.value = { id: sceneId, type: "scene" };
  uploadDialogVisible.value = true;
};

const selectFromLibrary = async (characterId: number) => {
  try {
    const result = await characterLibraryAPI.list({ page_size: 50 });
    libraryItems.value = result.items || [];
    currentUploadTarget.value = characterId;
    libraryDialogVisible.value = true;
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.loadLibraryFailed"));
  }
};

const addToCharacterLibrary = async (character: any) => {
  if (!character.image_url) {
    ElMessage.warning($t("workflow.generateImageFirst"));
    return;
  }

  try {
    await ElMessageBox.confirm(
      $t("workflow.addToLibraryConfirm", { name: character.name }),
      $t("workflow.addToLibrary"),
      {
        confirmButtonText: $t("common.confirm"),
        cancelButtonText: $t("common.cancel"),
        type: "info",
      },
    );

    await characterLibraryAPI.addCharacterToLibrary(character.id.toString());
    ElMessage.success($t("workflow.addedToLibrary"));
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || $t("workflow.addFailed"));
    }
  }
};

const selectLibraryItem = async (item: any) => {
  try {
    if (currentUploadTarget.value?.type === "character") {
      await characterLibraryAPI.applyFromLibrary(
        currentUploadTarget.value.id.toString(),
        item.id,
      );
      ElMessage.success("Character image applied");
      await loadDramaData();
      libraryDialogVisible.value = false;
    }
  } catch (error: any) {
    ElMessage.error(error.message || "Apply failed");
  }
};

const handleUploadSuccess = async (response: any) => {
  try {
    const imageUrl = response.url || response.data?.url;
    const localPath = response.local_path || response.data?.local_path;

    if (!imageUrl && !localPath) {
      ElMessage.error("Upload failed: missing image URL");
      return;
    }

    if (currentUploadTarget.value?.type === "character") {
      await characterLibraryAPI.updateCharacter(
        currentUploadTarget.value.id.toString(),
        {
          image_url: imageUrl,
          local_path: localPath,
        },
      );
      ElMessage.success("Uploaded");
    } else if (currentUploadTarget.value?.type === "outfit") {
      // For outfits, we need characterId (parentId) and outfitId (id)
      if (currentUploadTarget.value.parentId) {
        await characterLibraryAPI.updateOutfit(
          currentUploadTarget.value.parentId,
          currentUploadTarget.value.id,
          {
            image_url: imageUrl,
            local_path: localPath,
          },
        );
        ElMessage.success("Outfit image uploaded");
      }
    } else if (currentUploadTarget.value?.type === "scene") {
      // 更新场景图片
      await dramaAPI.updateScene(currentUploadTarget.value.id.toString(), {
        image_url: imageUrl,
        local_path: localPath,
      });
      ElMessage.success($t("workflow.sceneImageUploadSuccess"));
    }

    await loadDramaData();
    uploadDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error(error.message || "Upload failed");
  }
};

const handleUploadError = () => {
  ElMessage.error("Upload failed, please retry");
};

const deleteCharacter = async (characterId: number) => {
  try {
    await ElMessageBox.confirm(
      $t("workflow.deleteCharacterConfirm"),
      $t("workflow.deleteConfirmTitle"),
      {
        type: "warning",
        confirmButtonText: $t("workflow.confirmButtonText"),
        cancelButtonText: $t("workflow.cancelButtonText"),
      },
    );

    await characterLibraryAPI.deleteCharacter(characterId);
    ElMessage.success("Character deleted");
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const goToProfessionalUI = () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("Episode info not found");
    return;
  }

  router.push({
    name: "ProfessionalEditor",
    params: {
      dramaId: dramaId,
      episodeNumber: episodeNumber,
    },
  });
};

const goToCompose = () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("Episode info not found");
    return;
  }

  router.push({
    name: "SceneComposition",
    params: {
      id: dramaId,
      episodeId: currentEpisode.value.id,
    },
  });
};

// 打开添加场景对话框
const openAddSceneDialog = () => {
  newScene.value = {
    location: "",
    time: "",
    prompt: "",
    image_url: "",
    local_path: "",
  };
  addSceneDialogVisible.value = true;
};

// 保存场景
const saveScene = async () => {
  if (!newScene.value.location) {
    ElMessage.warning($t("workflow.pleaseEnterSceneName"));
    return;
  }

  if (!currentEpisode.value?.id) {
    ElMessage.error($t("workflow.chapterInfoNotExist"));
    return;
  }

  try {
    // 创建场景，关联到当前章节
    await dramaAPI.createScene({
      drama_id: parseInt(dramaId),
      episode_id: parseInt(currentEpisode.value.id),
      location: newScene.value.location,
      time: newScene.value.time || "",
      prompt: newScene.value.prompt,
      image_url: newScene.value.image_url,
      local_path: newScene.value.local_path,
    });

    ElMessage.success($t("workflow.sceneAddSuccess"));
    addSceneDialogVisible.value = false;

    // 重新加载数据以更新场景列表
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.sceneAddFailed"));
  }
};

// 处理场景图片上传成功
const handleSceneImageSuccess = (response: any) => {
  console.log("Scene image upload response:", response);

  // 处理不同的响应结构
  const imageUrl = response.url || response.data?.url;
  const localPath = response.local_path || response.data?.local_path;

  if (imageUrl) {
    newScene.value.image_url = imageUrl;
  }
  if (localPath) {
    newScene.value.local_path = localPath;
  }

  console.log("Updated newScene:", newScene.value);

  if (imageUrl || localPath) {
    ElMessage.success($t("workflow.imageUploadSuccess"));
  } else {
    ElMessage.warning($t("workflow.imageUploadSuccessNoUrl"));
  }
};

// 图片上传前的校验
const beforeAvatarUpload = (file: File) => {
  const isImage = file.type.startsWith("image/");
  const isLt10M = file.size / 1024 / 1024 < 10;

  if (!isImage) {
    ElMessage.error("Only image files are allowed");
    return false;
  }
  if (!isLt10M) {
    ElMessage.error("Image size cannot exceed 10MB");
    return false;
  }
  return true;
};

// 打开从剧本提取场景对话框
const openExtractSceneDialog = () => {
  extractScenesDialogVisible.value = true;
};

// 从剧本提取场景
const handleExtractScenes = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error($t("workflow.chapterInfoNotExist"));
    return;
  }

  try {
    extractingScenes.value = true;
    await dramaAPI.extractBackgrounds(currentEpisode.value.id.toString());

    ElMessage.success($t("workflow.sceneExtractSubmitted"));
    extractScenesDialogVisible.value = false;

    // 自动刷新几次
    let checkCount = 0;
    const maxChecks = 5;
    const checkInterval = setInterval(async () => {
      checkCount++;
      await loadDramaData();

      if (checkCount >= maxChecks) {
        clearInterval(checkInterval);
      }
    }, 3000);
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.sceneExtractFailed"));
  } finally {
    extractingScenes.value = false;
  }
};

// ── Storyboard batch first-frame prompt ──────────────────────────────────────

const loadEpisodeFramePrompts = async () => {
  const ep = currentEpisode.value;
  if (!ep?.id) return;
  try {
    const data = await getEpisodeFramePrompts(ep.id);
    episodeFramePrompts.value = data.frame_prompts_by_storyboard || {};
  } catch (error: any) {
    console.error("Failed to load episode frame prompts:", error);
  }
};

/** Latest video generation per storyboard (for workflow table status). */
const loadEpisodeVideos = async () => {
  try {
    const ep = currentEpisode.value;
    const targetStoryboardIds = new Set<number>(
      (ep?.storyboards || [])
        .map((s: any) => Number(s?.id))
        .filter((id: number) => !Number.isNaN(id)),
    );

    const allItems: VideoGeneration[] = [];
    let page = 1;
    const pageSize = 100;
    const maxPages = 80;
    let totalPages = 1;

    // Scan pages until we either read all pages or already have video data
    // for every storyboard in the current episode.
    do {
      const res = await videoAPI.listVideos({
        drama_id: dramaId,
        page,
        page_size: pageSize,
      });
      const items = res.items || [];
      allItems.push(...items);
      totalPages = Math.max(1, res.pagination?.total_pages ?? 1);

      if (targetStoryboardIds.size > 0) {
        const covered = new Set<number>();
        for (const v of allItems) {
          const sid = Number(v?.storyboard_id);
          if (!Number.isNaN(sid) && targetStoryboardIds.has(sid)) {
            covered.add(sid);
          }
        }
        if (covered.size >= targetStoryboardIds.size) break;
      }

      page++;
    } while (page <= totalPages && page <= maxPages);

    const sorted = [...allItems].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    );
    const map: Record<number, VideoGeneration> = {};
    const seen = new Set<number>();
    for (const v of sorted) {
      const sid = Number(v?.storyboard_id);
      if (Number.isNaN(sid) || seen.has(sid)) continue;
      seen.add(sid);
      map[sid] = v;
    }
    latestVideoByStoryboard.value = map;
  } catch (e) {
    console.warn("Failed to load episode videos:", e);
  }
};

const extractProviderFromModel = (modelName: string): string => {
  if (modelName.startsWith("doubao-") || modelName.startsWith("seedance")) {
    return "doubao";
  }
  if (modelName.startsWith("runway")) return "runway";
  if (modelName.startsWith("pika")) return "pika";
  if (
    modelName.startsWith("MiniMax-") ||
    modelName.toLowerCase().startsWith("minimax") ||
    modelName.startsWith("hailuo")
  ) {
    return "minimax";
  }
  if (modelName.startsWith("sora")) return "openai";
  if (modelName.startsWith("kling")) return "kling";
  return "doubao";
};

const storyboardVideoPrompt = (sb: any): string => {
  const ltx = sb?.ltx_video_prompt?.trim?.() ?? "";
  const vp = sb?.video_prompt?.trim?.() ?? "";
  const act = sb?.action?.trim?.() ?? "";
  return (ltx || vp || act || "").trim();
};

const hasFirstFramePrompt = (storyboardId: number | string): boolean => {
  const prompts = episodeFramePrompts.value[String(storyboardId)] || [];
  return prompts.some((p) => p.frame_type === "first");
};

const handleStoryboardSelectionChange = (rows: any[]) => {
  selectedStoryboardIds.value = rows.map((r) => r.id);
  const total = currentEpisode.value?.storyboards?.length || 0;
  selectAllStoryboards.value =
    total > 0 && selectedStoryboardIds.value.length === total;
};

const storyboardTableRef = ref<any>(null);

const toggleSelectAllStoryboards = (checked: boolean) => {
  const table = storyboardTableRef.value;
  if (!table) return;
  if (checked) {
    table.toggleAllSelection();
  } else {
    table.clearSelection();
  }
};

const pollTaskUntilDone = async (taskId: string): Promise<void> => {
  const maxAttempts = 60;
  const interval = 3000;
  for (let i = 0; i < maxAttempts; i++) {
    await new Promise((r) => setTimeout(r, interval));
    try {
      const task = await taskAPI.getStatus(taskId);
      if (task.status === "completed" || task.status === "failed") return;
    } catch {
      // continue polling on transient errors
    }
  }
};

const toNumericStoryboardId = (id: number | string): number =>
  typeof id === "string" ? parseInt(id, 10) : id;

const batchGenerateStoryboardImages = async () => {
  if (selectedStoryboardIds.value.length === 0) return;
  batchGeneratingStoryboardImages.value = true;
  let done = 0;
  let failed = 0;
  const ids = [...selectedStoryboardIds.value];
  try {
    for (const rawId of ids) {
      const storyboardId = toNumericStoryboardId(rawId);
      const sb = currentEpisode.value?.storyboards?.find(
        (s: any) => Number(s.id) === storyboardId,
      );
      if (!sb) {
        failed++;
        continue;
      }
      try {
        const fpData = await getStoryboardFramePrompts(storyboardId);
        const fp = fpData.frame_prompts?.find((p) => p.frame_type === "first");
        if (!fp?.prompt) {
          failed++;
          continue;
        }
        const referenceImages: string[] = [];
        if (sb.background?.local_path)
          referenceImages.push(sb.background.local_path);
        else if (sb.background?.image_url)
          referenceImages.push(sb.background.image_url);
        if (Array.isArray(sb.characters)) {
          sb.characters.forEach((char: any) => {
            if (char.local_path) referenceImages.push(char.local_path);
            else if (char.image_url) referenceImages.push(char.image_url);
          });
        }
        await imageAPI.generateImage({
          drama_id: dramaId,
          prompt: fp.prompt,
          storyboard_id: storyboardId,
          image_type: "storyboard",
          frame_type: "first",
          reference_images:
            referenceImages.length > 0 ? referenceImages : undefined,
          model: selectedImageModel.value || undefined,
        });
        done++;
      } catch {
        failed++;
      }
    }
    ElMessage.success(
      $t("workflow.batchShotImageSubmitted", { done, failed }),
    );
    await loadDramaData();
    let checkCount = 0;
    const maxChecks = 25;
    const iv = setInterval(async () => {
      checkCount++;
      await loadDramaData();
      if (checkCount >= maxChecks) clearInterval(iv);
    }, 3000);
  } catch (error: any) {
    ElMessage.error(
      error.message || $t("workflow.batchShotImageFailed"),
    );
  } finally {
    batchGeneratingStoryboardImages.value = false;
  }
};

const batchGenerateFirstFramePrompts = async () => {
  if (selectedStoryboardIds.value.length === 0) return;
  batchGeneratingFramePrompts.value = true;
  try {
    const results = await Promise.allSettled(
      selectedStoryboardIds.value.map((id) => generateFirstFrame(id)),
    );

    const taskIds: string[] = [];
    results.forEach((result, idx) => {
      if (result.status === "fulfilled") {
        taskIds.push(result.value.task_id);
      } else {
        console.error(
          `Failed to submit frame prompt for storyboard ${selectedStoryboardIds.value[idx]}:`,
          result.reason,
        );
      }
    });

    ElMessage.success($t("workflow.batchFramePromptSubmitted"));

    // Wait for all tasks to complete
    await Promise.allSettled(taskIds.map((id) => pollTaskUntilDone(id)));

    await loadEpisodeFramePrompts();
    ElMessage.success($t("workflow.batchFramePromptDone"));
  } catch (error: any) {
    ElMessage.error(error.message || "Batch frame prompt generation failed");
  } finally {
    batchGeneratingFramePrompts.value = false;
  }
};

// ── Batch Generate LTX video prompts ────────────────────────────────────────
const batchGenerateLtxVideoPrompts = async () => {
  if (selectedStoryboardIds.value.length === 0) return;

  batchGeneratingLtxVideoPrompts.value = true;
  selectedStoryboardIds.value.forEach((id) => {
    const sid = Number(id);
    if (!Number.isNaN(sid)) ltxVideoPromptGeneratingShots.value[sid] = true;
  });

  try {
    const ep = currentEpisode.value;
    if (!ep?.id) {
      ElMessage.error("Episode not found");
      return;
    }

    const res: any = await ltxVideoPromptAPI.batchGenerateLtxVideoPrompts(
      ep.id,
      selectedStoryboardIds.value,
      selectedTextModel.value || undefined,
    );

    const taskId = res?.task_id;
    if (!taskId) {
      ElMessage.error("Missing task_id from response");
      return;
    }

    ElMessage.success(
      $t("workflow.batchLtxVideoPromptSubmitted"),
    );

    await pollTaskUntilDone(taskId);

    // Reload to show per-shot ready status
    await loadDramaData();
    ElMessage.success($t("workflow.batchLtxVideoPromptDone"));
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchLtxVideoPromptFailed"));
  } finally {
    batchGeneratingLtxVideoPrompts.value = false;
    ltxVideoPromptGeneratingShots.value = {};
  }
};

// ── Batch generate videos (first-frame + LTX/video prompt, same as Professional Editor defaults) ──
const batchGenerateVideos = async () => {
  if (selectedStoryboardIds.value.length === 0) return;
  if (!selectedVideoModel.value) {
    ElMessage.warning($t("workflow.pleaseSelectVideoModel"));
    return;
  }

  batchGeneratingVideos.value = true;
  const ids = [...selectedStoryboardIds.value];
  ids.forEach((id) => {
    const sid = Number(id);
    if (!Number.isNaN(sid)) videoBatchSubmittingShots.value[sid] = true;
  });

  let done = 0;
  let failed = 0;

  try {
    const ep = currentEpisode.value;
    if (!ep?.storyboards?.length) {
      ElMessage.error($t("workflow.batchVideoNoEpisode"));
      return;
    }

    const byId = new Map<number, any>(
      ep.storyboards.map((s: any) => [Number(s.id), s]),
    );

    for (const rawId of ids) {
      const storyboardId = Number(rawId);
      const sb = byId.get(storyboardId);
      const prompt = sb ? storyboardVideoPrompt(sb) : "";
      if (!prompt || prompt.length < 5) {
        failed++;
        videoBatchSubmittingShots.value[storyboardId] = false;
        continue;
      }

      let first: ImageGeneration | undefined;
      try {
        const imgRes = await imageAPI.listImages({
          storyboard_id: storyboardId,
          frame_type: "first",
          page: 1,
          page_size: 30,
        });
        first = imgRes.items?.find(
          (i) =>
            i.status === "completed" && (i.image_url || i.local_path),
        );
      } catch {
        first = undefined;
      }

      if (!first) {
        failed++;
        videoBatchSubmittingShots.value[storyboardId] = false;
        continue;
      }

      const duration = Math.min(
        10,
        Math.max(4, Math.round(Number(sb?.duration) || 5)),
      );

      const requestParams: Record<string, unknown> = {
        drama_id: dramaId,
        storyboard_id: storyboardId,
        prompt,
        duration,
        provider: extractProviderFromModel(selectedVideoModel.value),
        model: selectedVideoModel.value,
        reference_mode: "single",
        aspect_ratio: drama.value?.aspect_ratio || "16:9",
        image_gen_id: first.id,
      };
      if (first.local_path) {
        requestParams.image_local_path = first.local_path;
      } else if (first.image_url) {
        requestParams.image_url = first.image_url;
      }

      try {
        await videoAPI.generateVideo(requestParams as any);
        done++;
      } catch {
        failed++;
      } finally {
        videoBatchSubmittingShots.value[storyboardId] = false;
      }
    }

    ElMessage.success($t("workflow.batchVideoSubmitted", { done, failed }));
    await loadEpisodeVideos();
    await loadDramaData();

    let checkCount = 0;
    const maxChecks = 40;
    const iv = setInterval(async () => {
      checkCount++;
      await loadEpisodeVideos();
      if (checkCount >= maxChecks) clearInterval(iv);
    }, 3000);
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchVideoFailed"));
  } finally {
    batchGeneratingVideos.value = false;
    videoBatchSubmittingShots.value = {};
  }
};

// ── Export shot images (zip: Ep…/drama/shot_…/) ───────────────────────────────

const sanitizeZipBaseName = (s: string) =>
  s.replace(/[/\\:*?"<>|]/g, "_").trim().slice(0, 100) || "export";

const exportShotImagesZip = async () => {
  const ep = currentEpisode.value;
  const d = drama.value;
  if (!ep?.storyboards?.length) {
    ElMessage.warning($t("workflow.exportShotImagesNoStoryboards"));
    return;
  }
  const shots = ep.storyboards as any[];
  const shotIds = new Set(shots.map((s) => Number(s.id)));

  exportingShotImages.value = true;
  try {
    const all: ImageGeneration[] = [];
    let page = 1;
    let totalPages = 1;
    const maxPages = 80;
    do {
      const res = await imageAPI.listImages({
        drama_id: dramaId,
        status: "completed",
        page,
        page_size: 100,
      });
      all.push(...res.items);
      totalPages = Math.max(1, res.pagination?.total_pages ?? 1);
      page++;
    } while (page <= totalPages && page <= maxPages);

    const relevant = all.filter(
      (g) =>
        g.storyboard_id != null &&
        shotIds.has(Number(g.storyboard_id)) &&
        (!g.image_type || g.image_type === "storyboard"),
    );
    const hasComposed = shots.some((s) => s.composed_image);
    if (relevant.length === 0 && !hasComposed) {
      ElMessage.warning($t("workflow.exportShotImagesEmpty"));
      return;
    }

    ElMessage.info($t("workflow.exportShotImagesBuilding"));
    const blob = await buildShotImagesZip({
      dramaTitle: d?.title || "drama",
      episodeNumber,
      episodeTitle: ep.title || `Episode_${episodeNumber}`,
      shots: shots.map((s) => ({
        id: s.id,
        storyboard_number: s.storyboard_number,
        title: s.title,
        composed_image: s.composed_image,
      })),
      imageGens: all,
    });

    const nameBase = sanitizeZipBaseName(
      `Ep${episodeNumber}_${d?.title || "drama"}_shot_images`,
    );
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${nameBase}.zip`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    ElMessage.success($t("workflow.exportShotImagesDone"));
  } catch (e: any) {
    ElMessage.error(e?.message || $t("workflow.exportShotImagesFailed"));
  } finally {
    exportingShotImages.value = false;
  }
};

// ── Export full episode video (client-side FFmpeg merge, same as timeline flow) ──

const exportFullEpisodeVideo = async () => {
  const ep = currentEpisode.value;
  if (!ep?.id || !ep.storyboards?.length) {
    ElMessage.warning($t("workflow.exportFullVideoNoStoryboards"));
    return;
  }

  exportingFullVideo.value = true;
  try {
    await loadEpisodeVideos();

    const ordered = [...ep.storyboards].sort(
      (a: any, b: any) =>
        Number(a?.storyboard_number || 0) - Number(b?.storyboard_number || 0),
    );

    const clips = ordered
      .map((sb: any) => {
        const sid = Number(sb?.id);
        const latest = latestVideoByStoryboard.value[sid];
        const videoUrl = getVideoUrl(latest);
        if (!latest || latest.status !== "completed" || !videoUrl) return null;

        const duration = Math.max(
          1,
          Number(latest?.duration) || Number(sb?.duration) || 5,
        );
        return {
          url: videoUrl,
          startTime: 0,
          endTime: duration,
          duration,
          transition: { type: "none" as const, duration: 0 },
        };
      })
      .filter(Boolean) as Array<{
      url: string;
      startTime: number;
      endTime: number;
      duration: number;
      transition: { type: string; duration: number };
    }>;

    if (clips.length === 0) {
      ElMessage.warning($t("workflow.exportFullVideoNoVideos"));
      return;
    }

    ElMessage.info($t("workflow.exportFullVideoSubmitted"));

    const mergedBlob = await videoMerger.mergeVideos(clips);
    const url = URL.createObjectURL(mergedBlob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `episode_${episodeNumber}_full.mp4`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    ElMessage.success($t("workflow.exportFullVideoDone"));
  } catch (e: any) {
    // If browser-side merge fails, fallback to backend merge path.
    try {
      const epId = String(ep.id);
      const ordered = [...ep.storyboards].sort(
        (a: any, b: any) =>
          Number(a?.storyboard_number || 0) - Number(b?.storyboard_number || 0),
      );
      const fallbackClips = ordered
        .map((sb: any, index: number) => {
          const sid = Number(sb?.id);
          const latest = latestVideoByStoryboard.value[sid];
          const videoUrl = getVideoUrl(latest);
          if (!latest || latest.status !== "completed" || !videoUrl) return null;
          const duration = Math.max(
            1,
            Number(latest?.duration) || Number(sb?.duration) || 5,
          );
          return {
            storyboard_id: String(sid),
            order: index,
            start_time: 0,
            end_time: duration,
            duration,
            transition: { type: "none", duration: 0 },
          };
        })
        .filter(Boolean);
      const result: any = await dramaAPI.finalizeEpisode(epId, {
        episode_id: epId,
        clips: fallbackClips,
      });
      if (result?.video_url) {
        const link = document.createElement("a");
        link.href = getVideoUrl({ video_url: result.video_url });
        link.download = `episode_${episodeNumber}_full.mp4`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        ElMessage.success($t("workflow.exportFullVideoDone"));
        return;
      }
      ElMessage.error(e?.message || $t("workflow.exportFullVideoFailed"));
    } catch {
      ElMessage.error(e?.message || $t("workflow.exportFullVideoFailed"));
    }
  } finally {
    exportingFullVideo.value = false;
  }
};

// 监听步骤变化，保存到 localStorage
watch(currentStep, (newStep) => {
  localStorage.setItem(getStepStorageKey(), newStep.toString());
});

onMounted(() => {
  loadDramaData();
  loadSavedModelConfig();
  loadAIConfigs();
});
</script>

<style scoped lang="scss">
/* ========================================
   Page Layout / 页面布局 - 紧凑边距
   ======================================== */
.page-container {
  min-height: 100vh;
  background: var(--bg-primary);
  // padding: var(--space-2) var(--space-3);
  transition: background var(--transition-normal);
}

@media (min-width: 768px) {
  .page-container {
    // padding: var(--space-3) var(--space-4);
  }
}

@media (min-width: 1024px) {
  .page-container {
    // padding: var(--space-4) var(--space-5);
  }
}

.content-wrapper {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  margin: 0 auto;
  width: 100%;
  height: 100vh;
  overflow: hidden;
}

.content-container {
  height: calc(100% - 134px);
  overflow-y: auto;
}

.actions-container {
  height: 70px;
  background: var(--bg-card);
  overflow: hidden;
}

/* Header styles matching PageHeader component */
.page-header {
  margin-bottom: var(--space-3);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-primary);
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex-shrink: 0;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.875rem;
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;

  &:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }
}

.nav-divider {
  width: 1px;
  height: 2rem;
  background: var(--border-primary);
}

.header-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.025em;
  line-height: 1.2;
  white-space: nowrap;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
}

.header-right {
  flex-shrink: 0;
}

.workflow-card {
  height: calc(100% - 24px);
  margin: 12px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  border: 1px solid var(--border-primary);

  :deep(.el-card__body) {
    padding: 0;
  }
}

.custom-steps {
  display: flex;
  align-items: center;
  gap: 12px;

  .step-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border-radius: 20px;
    background: var(--bg-card-hover);
    transition: all 0.3s;
    cursor: pointer;
    user-select: none;

    &:hover {
      background: var(--accent-light);
    }

    &.active {
      background: var(--accent-light);

      .step-circle {
        background: var(--accent);
        color: var(--text-inverse);
      }
    }

    &.current {
      background: var(--accent);
      color: var(--text-inverse);

      .step-circle {
        background: var(--bg-card);
        color: var(--accent);
      }

      .step-text {
        color: var(--text-inverse);
      }
    }

    .step-circle {
      width: 28px;
      height: 28px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--border-secondary);
      color: var(--text-secondary);
      font-weight: 600;
      transition: all 0.3s;
    }

    .step-text {
      font-size: 14px;
      font-weight: 500;
      white-space: nowrap;
    }
  }

  .step-arrow {
    color: var(--border-secondary);
  }
}

.stage-card {
  margin: 12px;

  &.stage-card-fullscreen {
    .stage-body-fullscreen {
      min-height: calc(100vh - 200px);
    }
  }
}

.stage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .header-info {
      h2 {
        margin: 0 0 4px 0;
        font-size: 20px;
      }

      p {
        margin: 0;
        color: var(--text-muted);
        font-size: 14px;
      }
    }
  }
}

.stage-body {
  background: var(--bg-card);
}

.shots-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;

  h3 {
    margin: 0;
  }
}

.shots-batch-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin: 12px 0;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
}

.action-buttons-inline {
  display: flex;
  gap: 12px;
}

.script-textarea {
  margin: 16px 0;

  &.script-textarea-fullscreen {
    :deep(textarea) {
      min-height: 500px;
      font-size: 14px;
      line-height: 1.8;
    }
  }
}

.image-gen-section {
  margin-bottom: 32px;

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 16px;
    background: var(--bg-secondary);
    // border-radius: 8px;
    // border: 1px solid var(--border-primary);

    .section-title {
      display: flex;
      align-items: center;
      gap: 16px;

      h3 {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--text-primary);

        .el-icon {
          color: var(--accent);
          font-size: 18px;
        }
      }

      .el-alert {
        border-radius: 4px;
      }
    }

    .section-actions {
      display: flex;
      align-items: center;
    }
  }
}

.empty-shots {
  padding: 60px 0;
  text-align: center;
}

.extracted-title {
  margin-bottom: 8px;
  color: var(--text-secondary);
}

.secondary-text {
  color: var(--text-muted);
  margin-left: 4px;
}

.task-message {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
}

.model-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}

.fixed-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
  box-shadow: var(--shadow-card);
  transition: all 0.2s;

  &:hover {
    box-shadow: var(--shadow-card-hover);
  }

  :deep(.el-card__body) {
    flex: 1;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  .card-header {
    padding: 14px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    display: flex;
    justify-content: space-between;
    align-items: center;

    .header-left {
      flex: 1;
      min-width: 0;

      h4 {
        margin: 0 0 4px 0;
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .el-tag {
        margin-top: 0;
      }
    }
  }

  .card-image-container {
    flex: 1;
    width: 100%;
    min-height: 200px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary);

    .char-image,
    .scene-image {
      width: 100%;
      height: 100%;
      position: relative;
      z-index: 1;

      .el-image {
        width: 100%;
        height: 100%;
        border-radius: 0;
      }
    }

    .char-placeholder,
    .scene-placeholder {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: var(--text-muted);
      padding: 20px;

      &.generating {
        color: var(--warning);
        background: var(--warning-light);

        .rotating {
          animation: rotating 2s linear infinite;
        }
      }

      &.failed {
        color: var(--error);
        background: var(--error-light);
      }
      position: relative;
      z-index: 1;

      .el-icon {
        opacity: 0.5;
      }

      span {
        margin-top: 10px;
        font-size: 12px;
      }
    }
  }

  .shot-outfit-text {
  font-size: 11px;
  color: var(--accent);
  font-style: italic;
  margin-left: 4px;
}

.outfit-assignments {
  margin-top: 16px;
  background: var(--bg-secondary);
  padding: 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-primary);
}

.outfit-assign-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.outfit-assign-row:last-child {
  margin-bottom: 0;
}

.char-name-label {
  font-weight: 600;
  min-width: 120px;
  color: var(--text-primary);
}

.char-actions-grid {
    padding: 10px;
    background: var(--bg-card);
    border-top: 1px solid var(--border-primary);
    display: flex;
    justify-content: center;
    gap: 8px;

    .el-button {
      margin: 0;
    }
  }
}

.character-image-list {
  padding: 16px 5px;
  display: flex;
  gap: 24px;
  overflow-x: auto;
  margin-top: 16px;
  
  // Custom scrollbar for character row
  &::-webkit-scrollbar {
    height: 8px;
  }
  &::-webkit-scrollbar-thumb {
    background: var(--border-primary);
    border-radius: 4px;
  }

  .character-group {
    flex-shrink: 0;
    width: 260px; /* Slightly smaller than scenes */
  }
}

.scene-image-list {
  padding: 5px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); /* Larger scenes */
  gap: 24px;
  margin-top: 16px;

  .scene-item {
    min-height: 400px;
  }
}

.character-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: var(--bg-card);
  padding: 16px;
  border-radius: 12px;
  border: 1px solid var(--border-primary);

  .character-item {
    width: 100%;
  }
}


.outfits-sub-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
  border-top: 1px dashed var(--border-primary);
  padding-top: 16px;
  width: 100%;

  .outfit-grid {
    display: grid;
    grid-template-columns: repeat(1, 1fr);
    gap: 12px;
  }

  .outfit-section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 10px;
    border-bottom: 1px dashed var(--border-primary);
    
    .header-label {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-secondary);
      display: flex;
      align-items: center;
      gap: 6px;

      .el-icon {
        color: var(--accent);
      }
    }
  }
  
  .outfit-grid {
    display: flex;
    gap: 16px;
    padding-bottom: 8px;
    overflow-x: auto;
  }

  .outfit-item {
    width: 220px;
    flex-shrink: 0;
  }
}

.premium-workflow-card {
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
  background: var(--bg-card);
  transition: all 0.3s;
  
  &:hover {
    border-color: var(--accent);
    box-shadow: var(--shadow-lg);
  }
}

.card-header-premium {
  padding: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-primary);
}

.char-meta {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  
  .char-name {
    font-weight: 600;
    font-size: 14px;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.card-actions-premium {
  padding: 10px;
  display: flex;
  justify-content: center;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-primary);
}

.outfit-card-mini-workflow {
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
  background: var(--bg-card);
  width: 100%;
}

.outfit-image-mini {
  position: relative;
  height: 160px;
  background: var(--bg-secondary);
  
  .el-image {
    width: 100%;
    height: 100%;
  }
}

.outfit-placeholder-mini {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.outfit-name-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
  color: white;
  font-size: 11px;
  padding: 4px 8px;
  font-weight: 500;
}

.outfit-mini-actions {
  padding: 6px 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  .action-btns {
    display: flex;
    gap: 4px;
  }
}

// 角色库选择对话框
.library-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  max-height: 500px;
  overflow-y: auto;
  padding: 8px;

  .library-item {
    cursor: pointer;
    border: 2px solid transparent;
    border-radius: 8px;
    overflow: hidden;
    transition: all 0.3s;

    &:hover {
      border-color: var(--accent);
      transform: translateY(-2px);
      box-shadow: var(--shadow-lg);
    }

    .el-image {
      width: 100%;
      height: 150px;
    }

    .library-item-name {
      padding: 8px;
      text-align: center;
      font-size: 12px;
      background: var(--bg-secondary);
      color: var(--text-primary);
    }
  }
}

.empty-library {
  padding: 40px 0;
}

// 上传区域
.upload-area {
  :deep(.el-upload-dragger) {
    width: 100%;
    height: 200px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }
}

// 旋转动画
@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* ========================================
   Dark Mode / 深色模式
   ======================================== */
:deep(.el-card) {
  background: var(--bg-card);
  border-color: var(--border-primary);
}

:deep(.el-card__header) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

:deep(.el-table) {
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-secondary);
  --el-table-tr-bg-color: var(--bg-card);
  --el-table-row-hover-bg-color: var(--bg-card-hover);
  --el-table-border-color: var(--border-primary);
  --el-table-text-color: var(--text-primary);
  background: var(--bg-card);
}

:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  background: var(--bg-card);
  border-color: var(--border-primary);
}

:deep(
  .el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell
) {
  background: var(--bg-secondary);
}

:deep(.el-table__header-wrapper th) {
  background: var(--bg-secondary) !important;
  color: var(--text-secondary);
}

:deep(.el-dialog) {
  background: var(--bg-card);
}

:deep(.el-dialog__header) {
  background: var(--bg-card);
}

:deep(.el-form-item__label) {
  color: var(--text-primary);
}

:deep(.el-input__wrapper) {
  background: var(--bg-secondary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

:deep(.el-input__inner) {
  color: var(--text-primary);
}

:deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

:deep(.el-select-dropdown) {
  background: var(--bg-elevated);
  border-color: var(--border-primary);
}

:deep(.el-upload-dragger) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

/* ========================================
   Step 2: Cinematic Premium Entity Cards (Master UX/UI)
   ======================================== */
.premium-entity-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 20px;
  margin-top: 16px;
}

.premium-entity-card {
  position: relative;
  width: 100%;
  aspect-ratio: 2 / 3;
  border-radius: 12px;
  overflow: hidden;
  background: #11141a; /* Dark sleek background for ungenerated state */
  border: 1px solid var(--border-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.25);
    border-color: var(--border-secondary);
  }

  /* Image or Placeholder */
  .entity-bg-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .entity-placeholder {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #e5eaf3;
    z-index: 1;
    
    .placeholder-text {
      font-size: 15px;
      font-weight: 500;
    }
  }

  /* Overlays */
  .entity-overlay-gradient {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 60%;
    background: linear-gradient(to top, rgba(0, 0, 0, 0.95) 0%, rgba(0, 0, 0, 0.6) 40%, rgba(0, 0, 0, 0) 100%);
    pointer-events: none;
    z-index: 2;
  }

  .entity-header {
    position: absolute;
    top: 12px;
    left: 12px;
    right: 12px;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    z-index: 3;
  }

  .top-left-tag {
    background: rgba(255, 255, 255, 0.15) !important;
    border: none !important;
    backdrop-filter: blur(8px);
    color: #fff !important;
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  .top-left-icon {
    color: #ffffff;
    font-size: 26px;
    filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5));
  }

  .top-right-group {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 6px;
  }

  .time-tag {
    background: rgba(0, 0, 0, 0.6) !important;
    border: 1px solid rgba(255, 255, 255, 0.1) !important;
    color: #409eff !important;
    display: flex;
    align-items: center;
    gap: 4px;
    backdrop-filter: blur(4px);
  }

  .entity-footer {
    position: absolute;
    bottom: 16px;
    left: 16px;
    right: 16px;
    z-index: 3;
  }

  .entity-title {
    color: #ffffff;
    font-size: 18px;
    font-weight: 800;
    line-height: 1.3;
    text-shadow: 0 2px 4px rgba(0, 0, 0, 0.8);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}

/* ========================================
   Unifying Step 1 with Management Card Styles
   ======================================== */
.portrait-optimized-card {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.card-image-container {
  position: relative;
  width: 100%;
  aspect-ratio: 3/4;
  overflow: hidden;
  background: var(--bg-secondary);
}

.portrait-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--transition-slow);
}

.card-image-container:hover .portrait-image {
  transform: scale(1.05);
}

.card-overlay-premium {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.9) 0%, rgba(0, 0, 0, 0.2) 60%, transparent 100%);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 12px;
  pointer-events: none;
}

.overlay-top {
  display: flex;
  justify-content: space-between;
  width: 100%;
}

.role-tag {
  backdrop-filter: blur(4px);
  background: rgba(0, 0, 0, 0.4) !important;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.overlay-bottom {
  .char-name, .scene-location {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 700;
    color: white;
    text-shadow: 0 2px 8px rgba(0, 0, 0, 0.8);
    text-align: left;
  }
}

.image-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: #11141a;
  color: #e5eaf3;
  z-index: 1;
}

.image-placeholder span {
  font-size: 15px;
  font-weight: 500;
}
</style>
