<template>
  <div class="page-container">
    <div class="content-wrapper animate-fade-in">
      <!-- Page Header / 页面头部 -->
      <AppHeader :fixed="false" :show-logo="false">
        <template #left>
          <el-button text @click="$router.back()" class="back-btn">
            <el-icon><ArrowLeft /></el-icon>
            <span>{{ $t("common.back") }}</span>
          </el-button>
          <div class="page-title">
            <h1>{{ drama?.title || "" }}</h1>
            <span class="subtitle">{{ $t("drama.management.overview") }}</span>
          </div>
        </template>
      </AppHeader>

      <!-- Tabs / 标签页 -->
      <div class="tabs-wrapper">
        <el-tabs v-model="activeTab" class="management-tabs">
          <!-- 项目概览 -->
          <el-tab-pane :label="$t('drama.management.overview')" name="overview">
            <div class="stats-grid">
              <StatCard
                :label="$t('drama.management.episodeStats')"
                :value="episodesCount"
                :icon="Document"
                icon-color="var(--accent)"
                icon-bg="var(--accent-light)"
                value-color="var(--accent)"
                :description="$t('drama.management.episodesCreated')"
              />
              <StatCard
                :label="$t('drama.management.characterStats')"
                :value="charactersCount"
                :icon="User"
                icon-color="var(--success)"
                icon-bg="var(--success-light)"
                value-color="var(--success)"
                :description="$t('drama.management.charactersCreated')"
              />
              <StatCard
                :label="$t('drama.management.sceneStats')"
                :value="scenesCount"
                :icon="Picture"
                icon-color="var(--warning)"
                icon-bg="var(--warning-light)"
                value-color="var(--warning)"
                :description="$t('drama.management.sceneLibraryCount')"
              />
              <StatCard
                :label="$t('drama.management.propStats')"
                :value="propsCount"
                :icon="Box"
                icon-color="var(--primary)"
                icon-bg="var(--primary-light)"
                value-color="var(--primary)"
                :description="$t('drama.management.propsCreated')"
              />
            </div>

            <!-- 引导卡片：无章节时显示 -->
            <el-alert
              v-if="episodesCount === 0"
              :title="$t('drama.management.startFirstEpisode')"
              type="info"
              :closable="false"
              style="margin-top: 20px"
            >
              <template #default>
                <p style="margin: 8px 0">
                  {{ $t("drama.management.noEpisodesYet") }}
                </p>
                <el-button
                  type="primary"
                  :icon="Plus"
                  @click="createNewEpisode"
                  style="margin-top: 8px"
                >
                  {{ $t("drama.management.createFirstEpisode") }}
                </el-button>
              </template>
            </el-alert>

            <el-card shadow="never" class="project-info-card">
              <template #header>
                <div class="card-header">
                  <h3 class="card-title">
                    {{ $t("drama.management.projectInfo") }}
                  </h3>
                  <el-tag :type="getStatusType(drama?.status)" size="small">{{
                    getStatusText(drama?.status)
                  }}</el-tag>
                </div>
              </template>
              <el-descriptions :column="2" border class="project-descriptions">
                <el-descriptions-item
                  :label="$t('drama.management.projectName')"
                >
                  <span class="info-value">{{ drama?.title }}</span>
                </el-descriptions-item>
                <el-descriptions-item :label="$t('common.createdAt')">
                  <span class="info-value">{{
                    formatDate(drama?.created_at)
                  }}</span>
                </el-descriptions-item>
                <el-descriptions-item
                  :label="$t('drama.management.projectDesc')"
                  :span="2"
                >
                  <span class="info-desc">{{
                    drama?.description || $t("drama.management.noDescription")
                  }}</span>
                </el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="project-info-card" style="margin-top: 20px">
              <template #header>
                <div class="card-header">
                  <h3 class="card-title">Story generator</h3>
                  <el-tag size="small" type="info">Narrative → episodes</el-tag>
                </div>
              </template>
              <p style="margin: 0 0 12px; color: var(--el-text-color-secondary); font-size: 13px">
                Text-only planning via <strong>AI Config (text)</strong>. Agent 1 saves graph, characters, and
                <code>base_image_prompt</code> only. Agent 2 adds beats/outfits/scenes. Agent 3 writes
                <code>script_content</code>. Character images are generated later from Go to Edit or Character Management.
              </p>
              
              <div v-if="!isIdeaEditing && drama?.narrative_idea" style="margin-bottom: 8px; display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--el-color-success); font-weight: 600;">
                <el-icon><Lock /></el-icon>
                <span>✓ Ý tưởng đã lưu & khóa (Idea Saved & Locked)</span>
              </div>
              <div v-else-if="narrativeLoading" style="margin-bottom: 8px; display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--el-color-warning); font-weight: 600;">
                <el-icon><Loading /></el-icon>
                <span>⚠️ Đang chạy Pipeline - Không thể chỉnh sửa (Pipeline Running - Locked)</span>
              </div>

              <el-input
                v-model="narrativeIdea"
                type="textarea"
                :rows="4"
                placeholder="Idea / hook (can be short)."
                :readonly="!isIdeaEditing && !!drama?.narrative_idea"
                :disabled="narrativeLoading"
                :style="!isIdeaEditing && !!drama?.narrative_idea ? 'background-color: var(--el-fill-color-light); opacity: 0.85;' : ''"
              />

              <div style="margin-top: 12px; display: flex; flex-wrap: wrap; gap: 8px">
                <el-button
                  v-if="isIdeaEditing || !drama?.narrative_idea"
                  type="primary"
                  :loading="savingNarrativeIdea"
                  :disabled="!narrativeIdea.trim() || narrativeLoading"
                  @click="saveNarrativeIdea"
                >
                  <el-icon style="margin-right: 4px"><DocumentChecked /></el-icon> Save Idea
                </el-button>

                <el-button
                  v-else
                  type="warning"
                  :disabled="narrativeLoading"
                  @click="handleEditIdea"
                >
                  <el-icon style="margin-right: 4px"><Edit /></el-icon> Edit Idea
                </el-button>

                <el-button
                  type="danger"
                  :loading="narrativeLoading && currentAgentTask === 0"
                  :disabled="narrativeLoading && currentAgentTask !== 0"
                  @click="runAgent(0)"
                >
                  Run Full Pipeline
                </el-button>

                <el-button
                  type="primary"
                  :loading="narrativeLoading && currentAgentTask === 1"
                  :disabled="narrativeLoading && currentAgentTask !== 1"
                  @click="runAgent(1)"
                >
                  Agent 1: Architect World & Characters
                </el-button>

                <el-button
                  type="warning"
                  :loading="narrativeLoading && currentAgentTask === 2"
                  :disabled="narrativeLoading && currentAgentTask !== 2"
                  @click="runAgent(2)"
                >
                  Agent 2: Build Beats & Outfits
                </el-button>
                
                <el-button
                  type="success"
                  :loading="narrativeLoading && currentAgentTask === 3"
                  :disabled="narrativeLoading && currentAgentTask !== 3"
                  @click="runAgent(3)"
                >
                  Agent 3: Design Markdown Scripts
                </el-button>

                <el-button
                  v-if="episodesCount > 0"
                  @click="$router.push(`/dramas/${route.params.id}/play`)"
                >
                  Play interactive
                </el-button>
              </div>

              <!-- Pipeline Status Feedback -->
              <div v-if="episodesCount > 0 || charactersCount > 0" style="margin-top: 16px; padding: 16px; background-color: var(--bg-secondary); border-radius: 8px; font-size: 13px; border: 1px solid var(--border-primary);">
                <div style="font-weight: 600; margin-bottom: 12px; font-size: 14px;">Pipeline Data (Preview):</div>

                <!-- Global Storyline & Hook -->
                <div v-if="drama?.description || drama?.narrative_idea" style="margin-bottom: 16px; background: var(--bg-card); border-radius: 6px; padding: 12px 16px; border-left: 4px solid var(--el-color-primary); box-shadow: var(--shadow-sm);">
                  <div style="font-size: 11px; font-weight: 700; color: var(--el-color-primary); letter-spacing: 0.8px; margin-bottom: 6px; text-transform: uppercase; display: flex; align-items: center; gap: 6px;">
                    <el-icon><Film /></el-icon>
                    <span>🎬 GLOBAL STORYLINE & HOOK</span>
                  </div>
                  <div style="font-size: 13px; color: var(--text-secondary); line-height: 1.6; white-space: pre-wrap; font-style: italic;">
                    "{{ drama.description || drama.narrative_idea }}"
                  </div>
                </div>
                
                <div v-if="charactersCount > 0" style="margin-bottom: 12px;">
                  <div style="color: var(--text-secondary); margin-bottom: 4px;">
                    <el-icon><User /></el-icon> Characters ({{ charactersCount }}) - <em>Agent 1 data, images manual</em>:
                  </div>
                  <div style="display: flex; flex-wrap: wrap; gap: 6px;">
                    <el-tag v-for="c in drama?.characters" :key="c.id" size="small" type="success" effect="plain">{{ c.name }}</el-tag>
                  </div>
                </div>

                <div v-if="episodesCount > 0" style="margin-bottom: 12px;">
                  <div style="color: var(--text-secondary); margin-bottom: 4px;">
                    <el-icon><Document /></el-icon> Episodes ({{ episodesCount }}) - <em>Built by Agent 1</em>:
                  </div>
                  <div style="display: flex; flex-wrap: wrap; gap: 6px;">
                    <el-tag v-for="ep in sortedEpisodes" :key="ep.id" size="small" type="info" effect="plain">{{ ep.narrative_node_id }}: {{ ep.title }}</el-tag>
                  </div>
                </div>
                
                <div v-if="scenesCount > 0" style="margin-bottom: 8px;">
                  <div style="color: var(--text-secondary); margin-bottom: 4px;">
                    <el-icon><Picture /></el-icon> Scenes ({{ scenesCount }}) - <em>Built by Agent 2</em>:
                  </div>
                  <div style="display: flex; flex-wrap: wrap; gap: 6px;">
                    <el-tag v-for="sc in sortedScenes" :key="sc.id" size="small" type="warning" effect="plain">{{ sc.location }}</el-tag>
                  </div>
                </div>
                
                <div v-if="sortedEpisodes.some(ep => ep.script_content)" style="margin-top: 12px;">
                  <el-alert type="success" :closable="false" show-icon>
                    <strong>Agent 3 Complete:</strong> Markdown scripts have been generated. Go to Episodes tab and click Edit to read them.
                  </el-alert>
                </div>
                <div v-else-if="episodesCount > 0" style="margin-top: 12px;">
                  <el-alert type="info" :closable="false" show-icon>
                    <strong>Text planning in progress:</strong> Agent 1 skeleton episodes do not contain Episode Content yet. Run Agent 2 then Agent 3, or run Full Pipeline.
                  </el-alert>
                </div>
              </div>
            </el-card>
          </el-tab-pane>

          <!-- 章节管理 -->
          <el-tab-pane :label="$t('drama.management.episodes')" name="episodes">
            <div class="tab-header">
              <h2>{{ $t("drama.management.episodeList") }}</h2>
              <el-button
                type="primary"
                :icon="Plus"
                @click="createNewEpisode"
                >{{ $t("drama.management.createNewEpisode") }}</el-button
              >
            </div>

            <!-- 空状态引导 -->
            <el-empty
              v-if="episodesCount === 0"
              :description="$t('drama.management.noEpisodes')"
              style="margin-top: 40px"
            >
              <template #image>
                <el-icon :size="80" class="empty-icon"><Document /></el-icon>
              </template>
              <el-button type="primary" :icon="Plus" @click="createNewEpisode">
                {{ $t("drama.management.createFirstEpisode") }}
              </el-button>
            </el-empty>

            <template v-else>
              <NarrativeStoryGraph
                :source="narrativeMermaidSource"
                title="Story graph"
                emptyText="No branching data"
              />

              <div class="episode-batch-toolbar">
                <el-checkbox
                  v-model="episodeSelectAll"
                  @change="onEpisodeSelectAllChange"
                >
                  Select all
                </el-checkbox>
                <el-button
                  type="primary"
                  :loading="autoPipelineRunning"
                  :disabled="selectedEpisodesRows.length === 0"
                  @click="startAutoPipeline"
                >
                  Full auto production
                </el-button>
                <el-button
                  :disabled="!autoPipelineRunning"
                  @click="cancelAutoPipeline"
                >
                  Stop
                </el-button>
              </div>

              <el-table
                ref="episodeTableRef"
                :data="sortedEpisodes"
                border
                stripe
                row-key="id"
                style="margin-top: 12px"
                @selection-change="onEpisodeSelectionChange"
              >
                <el-table-column type="selection" width="48" reserve-selection />
                <el-table-column type="expand" label="Info" width="65">
                  <template #default="{ row }">
                    <div style="padding: 12px 16px; background: var(--bg-secondary); border-radius: 6px;">
                      <!-- Plot Outline (Agent 1 Architect) -->
                      <div v-if="row.state_snapshot?.plot_summary || (!row.state_snapshot?.timeline && row.description)" style="margin-bottom: 12px;">
                        <div style="font-size: 11px; font-weight: 700; color: var(--el-color-primary); letter-spacing: 0.8px; margin-bottom: 6px;">📖 PLOT OUTLINE (Agent 1 Architect)</div>
                        <div style="white-space: pre-wrap; font-size: 13px; color: var(--text-secondary); line-height: 1.6; background: var(--bg-card); border-radius: 4px; padding: 10px; border-left: 3px solid var(--el-color-primary);">
                          {{ row.state_snapshot?.plot_summary || row.description }}
                        </div>
                      </div>

                      <!-- Micro-beats (Agent 2 Builder) -->
                      <div v-if="row.state_snapshot?.timeline && row.description" style="margin-bottom: 12px;">
                        <div style="font-size: 11px; font-weight: 700; color: var(--accent); letter-spacing: 0.8px; margin-bottom: 6px;">🎬 MICRO-BEATS (Agent 2 Builder)</div>
                        <div style="white-space: pre-wrap; font-size: 13px; color: var(--text-secondary); line-height: 1.6; background: var(--bg-card); border-radius: 4px; padding: 10px; border-left: 3px solid var(--accent);">
                          {{ row.description }}
                        </div>
                      </div>

                      <!-- State Snapshot (Agent 2 State Tracking) -->
                      <div v-if="row.state_snapshot && row.state_snapshot.timeline">
                        <div style="font-size: 11px; font-weight: 700; color: var(--warning); letter-spacing: 0.8px; margin-bottom: 6px;">📋 STATE SNAPSHOT (Agent 2 State Tracking)</div>
                        <div style="font-size: 13px; color: var(--text-secondary); background: var(--bg-card); border-radius: 4px; padding: 10px; border-left: 3px solid var(--warning);">
                          <div v-if="row.state_snapshot.timeline" style="margin-bottom: 4px;"><strong>Timeline:</strong> {{ row.state_snapshot.timeline }}</div>
                          <div v-if="row.state_snapshot.character_statuses" style="margin-bottom: 4px;"><strong>Characters:</strong> {{ row.state_snapshot.character_statuses }}</div>
                          <div v-if="row.state_snapshot.key_items_locations"><strong>Key Items:</strong> {{ row.state_snapshot.key_items_locations }}</div>
                        </div>
                      </div>

                      <div v-if="!row.description && !row.state_snapshot" style="font-size: 13px; color: var(--text-muted); font-style: italic;">
                        No planning data available yet. Run the story pipeline to populate.
                      </div>
                    </div>
                  </template>
                </el-table-column>
              <el-table-column
                type="index"
                :label="$t('storyboard.table.number')"
                width="80"
              />
              <el-table-column
                prop="title"
                :label="$t('drama.management.episodeList')"
                min-width="180"
              />
              <el-table-column
                label="Plot Outline"
                min-width="280"
              >
                <template #default="{ row }">
                  <el-tooltip
                    v-if="row.state_snapshot?.plot_summary || row.description"
                    :content="row.state_snapshot?.plot_summary || row.description"
                    placement="top"
                    :show-after="300"
                    effect="dark"
                    :popper-style="{ maxWidth: '400px', whiteSpace: 'pre-wrap', lineHeight: '1.6', fontSize: '12px' }"
                  >
                    <div class="plot-outline-cell">
                      {{ row.state_snapshot?.plot_summary || row.description }}
                    </div>
                  </el-tooltip>
                  <span v-else class="plot-outline-empty">—</span>
                </template>
              </el-table-column>
              <el-table-column label="Node" width="88">
                <template #default="{ row }">
                  <span v-if="row.narrative_node_id">{{ row.narrative_node_id }}</span>
                  <span v-else>—</span>
                </template>
              </el-table-column>
              <el-table-column label="Branch" width="88">
                <template #default="{ row }">
                  <el-tag v-if="episodeChoiceCount(row)" type="warning" size="small">
                    {{ episodeChoiceCount(row) }}
                  </el-tag>
                  <span v-else>—</span>
                </template>
              </el-table-column>
              <!-- Micro-beats status badge -->
              <el-table-column label="Beats" width="76">
                <template #default="{ row }">
                  <el-tag v-if="row.description" type="success" size="small" effect="dark">✓</el-tag>
                  <el-tag v-else type="info" size="small" effect="plain">—</el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="$t('common.status')" width="120">
                <template #default="{ row }">
                  <el-tag :type="getEpisodeStatusType(row)">{{
                    getEpisodeStatusText(row)
                  }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="Shots" width="100">
                <template #default="{ row }">
                  {{ row.storyboards?.length || 0 }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('common.createdAt')" width="180">
                <template #default="{ row }">
                  {{ formatDate(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column
                :label="$t('storyboard.table.operations')"
                width="240"
                fixed="right"
                align="center"
                header-align="center"
              >
                <template #default="{ row }">
                  <div class="episode-ops-cell">
                    <el-button
                      size="small"
                      type="primary"
                      @click="enterEpisodeWorkflow(row)"
                    >
                      {{ $t("drama.management.goToEdit") }}
                    </el-button>
                    <el-button
                      size="small"
                      type="danger"
                      @click="deleteEpisode(row)"
                    >
                      {{ $t("common.delete") }}
                    </el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            </template>
          </el-tab-pane>

          <el-dialog
            v-model="autoPipelineDialogVisible"
            title="Auto production"
            width="560px"
            :close-on-click-modal="false"
            class="auto-pipeline-dialog"
          >
            <div
              v-if="autoPipelineQueueStatus !== 'idle'"
              class="auto-pipeline-status-bar"
              :data-state="autoPipelineQueueStatus"
            >
              <span class="auto-pipeline-status-text">{{
                autoPipelineStatusLine
              }}</span>
              <span
                v-if="autoPipelineRunning"
                class="auto-pipeline-pct"
                >{{ autoPipelineProgressPct }}%</span
              >
            </div>
            <el-progress
              v-if="autoPipelineRunning"
              class="auto-pipeline-progress"
              :percentage="autoPipelineProgressPct"
              :indeterminate="autoPipelineProgressPct === 0"
            />
            <div class="auto-pipeline-log">
              <div
                v-for="(line, idx) in autoPipelineLogLines"
                :key="idx"
                class="log-line"
              >
                {{ line }}
              </div>
            </div>
            <template #footer>
              <el-button @click="autoPipelineDialogVisible = false"
                >Close</el-button
              >
              <el-button
                v-if="autoPipelineLastFailure"
                type="primary"
                @click="openFailedEpisodeWorkflow"
              >
                Open episode to fix
              </el-button>
            </template>
          </el-dialog>

          <!-- 角色管理 -->
          <el-tab-pane
            :label="$t('drama.management.characters')"
            name="characters"
          >
            <div class="tab-header">
              <h2>{{ $t("drama.management.characterList") }}</h2>
              <div style="display: flex; gap: 10px">
                <el-button
                  type="success"
                  :icon="MagicStick"
                  :loading="batchGeneratingCharacterImages"
                  :disabled="charactersNeedingImages.length === 0"
                  @click="batchGenerateCharacterImages"
                >
                  Generate Missing Images ({{ charactersNeedingImages.length }})
                </el-button>
                <el-button
                  :icon="MagicStick"
                  @click="openExtractCharacterDialog"
                  >✨ Extract & Sync Entities</el-button
                >
                <el-button
                  type="primary"
                  :icon="Plus"
                  @click="openAddCharacterDialog"
                  >{{ $t("character.add") }}</el-button
                >
              </div>
            </div>

            <el-row :gutter="16" style="margin-top: 16px">
              <el-col
                :span="6"
                v-for="character in drama?.characters"
                :key="character.id"
                style="margin-bottom: 24px"
              >
                <el-card shadow="hover" class="character-card portrait-optimized-card" :body-style="{ padding: '0px' }">
                  <div class="card-image-container" :style="{ aspectRatio: cardAspectRatio }">
                    <el-image
                      v-if="character.local_path || character.image_url"
                      :src="getImageUrl(character)"
                      :alt="character.name"
                      class="portrait-image"
                      fit="cover"
                      :preview-src-list="[getImageUrl(character)]"
                      hide-on-click-modal
                      preview-teleported
                    />
                    <div v-else class="image-placeholder">
                      <el-icon :size="48"><User /></el-icon>
                      <span>{{ $t("common.notGenerated") }}</span>
                    </div>
                    
                    <div class="card-overlay-premium">
                      <div class="overlay-top" style="display: flex; justify-content: flex-start; width: 100%;">
                        <el-tag
                          :type="character.role === 'main' ? 'danger' : 'info'"
                          size="small"
                          effect="dark"
                          class="role-tag"
                        >
                          {{ character.role === "main" ? "Main" : character.role === "supporting" ? "Supporting" : "Minor" }}
                        </el-tag>
                      </div>
                      <div class="overlay-bottom">
                        <h4 class="char-name">{{ character.name }}</h4>
                      </div>
                    </div>
                  </div>

                  <div class="character-content">
                    <div class="char-description" v-if="character.appearance || character.description">
                      {{ character.appearance || character.description }}
                    </div>

                    <!-- Enhanced Outfits Section -->
                    <div v-if="character.outfits && character.outfits.length > 0" class="outfits-section">
                      <div class="outfits-header">
                        <span>{{ $t("character.outfits") || "Outfits" }} ({{ character.outfits.length }})</span>
                      </div>
                      <div class="outfits-scroll">
                        <div v-for="outfit in character.outfits" :key="outfit.id" class="outfit-card-mini">
                          <el-image 
                            :src="getImageUrl(outfit)" 
                            fit="cover" 
                            class="outfit-img"
                            :preview-src-list="[getImageUrl(outfit)]"
                            preview-teleported
                          />
                          <span class="outfit-name-tag">{{ outfit.name }}</span>
                          <div v-if="outfit.appearances" class="outfit-appearances-box" style="margin-top: 4px; display: flex; flex-wrap: wrap; gap: 2px; justify-content: center; width: 100%;">
                            <span 
                              v-for="node in outfit.appearances.split(',')" 
                              :key="node" 
                              style="font-size: 9px; padding: 1px 4px; background: rgba(64, 158, 255, 0.15); border: 1px solid rgba(64, 158, 255, 0.3); border-radius: 4px; color: #409eff; font-weight: bold; line-height: 1.1;"
                            >
                              {{ node }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="character-actions-premium">
                    <el-button-group>
                      <el-button size="small" :icon="Edit" @click="editCharacter(character)">{{ $t("common.edit") }}</el-button>
                      <el-button size="small" type="primary" :icon="MagicStick" @click="generateCharacterImage(character)">
                        {{ $t("prop.generateImage") }}
                      </el-button>
                      <el-button size="small" :icon="Box" @click="openOutfitsDialog(character)">
                        Outfits
                      </el-button>
                    </el-button-group>
                    <el-button size="small" type="danger" :icon="Delete" @click="deleteCharacter(character)" class="delete-btn" />
                  </div>
                </el-card>
              </el-col>
            </el-row>

            <el-empty
              v-if="!drama?.characters || drama.characters.length === 0"
              :description="$t('drama.management.noCharacters')"
            />
          </el-tab-pane>

          <!-- 场景库管理 -->
          <el-tab-pane :label="$t('drama.management.sceneList')" name="scenes">
            <div class="tab-header">
              <h2>{{ $t("drama.management.sceneList") }}</h2>
            </div>

            <el-row :gutter="16" style="margin-top: 16px">
              <el-col
                v-for="scene in sortedScenes"
                :key="scene.id"
                :span="6"
                style="margin-bottom: 24px"
              >
                <el-card :body-style="{ padding: '0px' }" class="portrait-optimized-card scene-card">
                  <div class="card-image-container" :style="{ aspectRatio: cardAspectRatio }">
                    <el-image
                      v-if="scene.local_path || scene.image_url"
                      :src="getImageUrl(scene)"
                      :alt="scene.location"
                      class="portrait-image"
                      fit="cover"
                      :preview-src-list="[getImageUrl(scene)]"
                      hide-on-click-modal
                      preview-teleported
                    />
                    <div v-else class="image-placeholder">
                      <el-icon :size="48"><Location /></el-icon>
                      <span>{{ $t("common.notGenerated") }}</span>
                    </div>
                    
                    <div class="card-overlay-premium">
                      <div class="overlay-top" style="display: flex; justify-content: space-between; width: 100%;">
                        <div style="color: white; font-size: 24px; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5));">
                          <el-icon><LocationInformation /></el-icon>
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

                  <div class="scene-actions">
                    <el-button size="small" @click="editScene(scene)">{{ $t("common.edit") }}</el-button>
                    <el-button size="small" type="primary" @click="generateSceneImage(scene.id)">
                      <el-icon style="margin-right: 4px"><MagicStick /></el-icon> {{ $t("prop.generateImage") }}
                    </el-button>
                    <el-button size="small" type="danger" @click="deleteScene(scene)">
                      <el-icon style="margin-right: 4px"><Delete /></el-icon> {{ $t("common.delete") }}
                    </el-button>
                  </div>
                </el-card>
              </el-col>
            </el-row>

            <el-empty
              v-if="scenes.length === 0"
              :description="$t('drama.management.noScenes')"
            />
          </el-tab-pane>

          <!-- 道具管理 -->
          <el-tab-pane :label="$t('drama.management.propList')" name="props">
            <div class="tab-header">
              <h2>{{ $t("drama.management.propList") }}</h2>
              <div style="display: flex; gap: 10px">
                <el-button :icon="Document" @click="openExtractDialog">{{
                  $t("prop.extract")
                }}</el-button>
                <el-button
                  type="primary"
                  :icon="Plus"
                  @click="openAddPropDialog"
                  >{{ $t("common.add") }}</el-button
                >
              </div>
            </div>

            <el-row :gutter="16" style="margin-top: 16px">
              <el-col :span="6" v-for="prop in drama?.props" :key="prop.id">
                <el-card shadow="hover" class="scene-card">
                  <div class="scene-preview">
                    <ImagePreview
                      :image-url="getImageUrl(prop)"
                      :alt="prop.name"
                      :size="120"
                      :show-placeholder-text="false"
                    />
                  </div>

                  <div class="scene-info">
                    <h4>{{ prop.name }}</h4>
                    <el-tag size="small" v-if="prop.type">{{
                      prop.type
                    }}</el-tag>
                    <p class="desc">{{ prop.description || prop.prompt }}</p>
                  </div>

                  <div class="scene-actions">
                    <el-button size="small" @click="editProp(prop)">{{
                      $t("common.edit")
                    }}</el-button>
                    <el-button
                      size="small"
                      @click="generatePropImage(prop)"
                      :disabled="!prop.prompt"
                      >{{ $t("prop.generateImage") }}</el-button
                    >
                    <el-button
                      size="small"
                      type="danger"
                      @click="deleteProp(prop)"
                      >{{ $t("common.delete") }}</el-button
                    >
                  </div>
                </el-card>
              </el-col>
            </el-row>

            <el-empty
              v-if="!drama?.props || drama.props.length === 0"
              :description="$t('drama.management.noProps')"
            />
          </el-tab-pane>
        </el-tabs>
      </div>

      <!-- 添加/编辑角色对话框 -->
      <el-dialog
        v-model="addCharacterDialogVisible"
        :title="editingCharacter ? $t('character.edit') : $t('character.add')"
        width="600px"
      >
        <el-form :model="newCharacter" label-width="100px">
          <el-form-item :label="$t('character.image')">
            <el-upload
              class="avatar-uploader"
              :action="`/api/v1/upload/image`"
              :show-file-list="false"
              :on-success="handleCharacterAvatarSuccess"
              :before-upload="beforeAvatarUpload"
            >
              <img
                v-if="hasImage(newCharacter)"
                :src="getImageUrl(newCharacter)"
                class="avatar"
                style="width: 100px; height: 100px; object-fit: cover"
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
                  width: 100px;
                  height: 100px;
                  font-size: 28px;
                  color: #8c939d;
                  text-align: center;
                  line-height: 100px;
                "
                ><Plus
              /></el-icon>
            </el-upload>
          </el-form-item>
          <el-form-item :label="$t('character.name')">
            <el-input
              v-model="newCharacter.name"
              :placeholder="$t('character.name')"
            />
          </el-form-item>
          <el-form-item :label="$t('character.role')">
            <el-select
              v-model="newCharacter.role"
              :placeholder="$t('common.pleaseSelect')"
            >
              <el-option label="Main" value="main" />
              <el-option label="Supporting" value="supporting" />
              <el-option label="Minor" value="minor" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('character.appearance')">
            <el-input
              v-model="newCharacter.appearance"
              type="textarea"
              :rows="3"
              :placeholder="$t('character.appearance')"
            />
          </el-form-item>
          <el-form-item label="Base Face Prompt">
            <el-input
              v-model="newCharacter.base_image_prompt"
              type="textarea"
              :rows="3"
              placeholder="Base face prompt generated by AI (for image gen)"
            />
          </el-form-item>
          <el-form-item :label="$t('character.personality')">
            <el-input
              v-model="newCharacter.personality"
              type="textarea"
              :rows="3"
              :placeholder="$t('character.personality')"
            />
          </el-form-item>
          <el-form-item :label="$t('character.description')">
            <el-input
              v-model="newCharacter.description"
              type="textarea"
              :rows="3"
              :placeholder="$t('common.description')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="addCharacterDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveCharacter">{{
            $t("common.confirm")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 添加/编辑场景对话框 -->
      <el-dialog
        v-model="addSceneDialogVisible"
        :title="editingScene ? $t('common.edit') : $t('common.add')"
        width="600px"
      >
        <el-form :model="newScene" label-width="100px">
          <el-form-item :label="$t('common.image')">
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
          <el-form-item :label="$t('common.name')">
            <el-input
              v-model="newScene.location"
              :placeholder="$t('common.name')"
            />
          </el-form-item>
          <el-form-item :label="$t('common.description')">
            <el-input
              v-model="newScene.prompt"
              type="textarea"
              :rows="4"
              :placeholder="$t('common.description')"
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

      <!-- 添加/编辑道具对话框 -->
      <el-dialog
        v-model="addPropDialogVisible"
        :title="editingProp ? $t('common.edit') : $t('common.add')"
        width="600px"
      >
        <el-form :model="newProp" label-width="100px">
          <el-form-item :label="$t('common.image')">
            <el-upload
              class="avatar-uploader"
              :action="`/api/v1/upload/image`"
              :show-file-list="false"
              :on-success="handlePropImageSuccess"
              :before-upload="beforeAvatarUpload"
            >
              <img
                v-if="hasImage(newProp)"
                :src="getImageUrl(newProp)"
                class="avatar"
                style="width: 100px; height: 100px; object-fit: cover"
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
                  width: 100px;
                  height: 100px;
                  font-size: 28px;
                  color: #8c939d;
                  text-align: center;
                  line-height: 100px;
                "
                ><Plus
              /></el-icon>
            </el-upload>
          </el-form-item>
          <el-form-item :label="$t('prop.name')">
            <el-input v-model="newProp.name" :placeholder="$t('prop.name')" />
          </el-form-item>
          <el-form-item :label="$t('prop.type')">
            <el-input
              v-model="newProp.type"
              :placeholder="$t('prop.typePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('prop.description')">
            <el-input
              v-model="newProp.description"
              type="textarea"
              :rows="3"
              :placeholder="$t('prop.description')"
            />
          </el-form-item>
          <el-form-item :label="$t('prop.prompt')">
            <el-input
              v-model="newProp.prompt"
              type="textarea"
              :rows="3"
              :placeholder="$t('prop.promptPlaceholder')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="addPropDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveProp">{{
            $t("common.confirm")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 从剧本提取道具对话框 -->
      <el-dialog
        v-model="extractPropsDialogVisible"
        :title="$t('prop.extractTitle')"
        width="500px"
      >
        <el-form label-width="100px">
          <el-form-item :label="$t('prop.selectEpisode')">
            <el-select
              v-model="selectedExtractEpisodeId"
              :placeholder="$t('common.pleaseSelect')"
              style="width: 100%"
            >
              <el-option
                v-for="ep in sortedEpisodes"
                :key="ep.id"
                :label="ep.title"
                :value="ep.id"
              />
            </el-select>
          </el-form-item>
          <el-alert
            :title="$t('prop.extractTip')"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form>
        <template #footer>
          <el-button @click="extractPropsDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            type="primary"
            @click="handleExtractProps"
            :disabled="!selectedExtractEpisodeId"
            >{{ $t("prop.startExtract") }}</el-button
          >
        </template>
      </el-dialog>

      <!-- 从剧本提取角色对话框 -->
      <el-dialog
        v-model="extractCharactersDialogVisible"
        :title="$t('prop.extractTitle')"
        width="500px"
      >
        <el-form label-width="100px">
          <el-form-item :label="$t('prop.selectEpisode')">
            <el-select
              v-model="selectedExtractEpisodeId"
              :placeholder="$t('common.pleaseSelect')"
              style="width: 100%"
            >
              <el-option
                v-for="ep in sortedEpisodes"
                :key="ep.id"
                :label="ep.title"
                :value="ep.id"
              />
            </el-select>
          </el-form-item>
          <el-alert
            :title="$t('prop.extractTip')"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form>
        <template #footer>
          <el-button @click="extractCharactersDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            type="primary"
            @click="handleExtractCharacters"
            :disabled="!selectedExtractEpisodeId"
            >{{ $t("prop.startExtract") }}</el-button
          >
        </template>
      </el-dialog>

      <!-- 从剧本提取场景对话框 -->
      <el-dialog
        v-model="extractScenesDialogVisible"
        :title="$t('prop.extractTitle')"
        width="500px"
      >
        <el-form label-width="100px">
          <el-form-item :label="$t('prop.selectEpisode')">
            <el-select
              v-model="selectedExtractEpisodeId"
              :placeholder="$t('common.pleaseSelect')"
              style="width: 100%"
            >
              <el-option
                v-for="ep in sortedEpisodes"
                :key="ep.id"
                :label="ep.title"
                :value="ep.id"
              />
            </el-select>
          </el-form-item>
          <el-alert
            :title="$t('prop.extractTip')"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form>
        <template #footer>
          <el-button @click="extractScenesDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            type="primary"
            @click="handleExtractScenes"
            :disabled="!selectedExtractEpisodeId"
            >{{ $t("prop.startExtract") }}</el-button
          >
        </template>
      </el-dialog>

      <!-- Manage Outfits Dialog -->
      <el-dialog
        v-model="outfitsDialogVisible"
        :title="`Manage Outfits: ${currentCharacterForOutfits?.name}`"
        width="700px"
      >
        <div style="display: flex; gap: 16px;">
          <!-- Left side: Form -->
          <div style="flex: 1;">
            <h4>{{ editingOutfit ? 'Edit Outfit' : 'New Outfit' }}</h4>
            <el-form label-position="top">
              <el-form-item label="Outfit Name">
                <el-input v-model="newOutfit.name" placeholder="e.g. Prison Uniform, Prom Dress" />
              </el-form-item>
              <el-form-item label="Appearance Prompt">
                <el-input v-model="newOutfit.prompt" type="textarea" :rows="3" placeholder="Describe the outfit..." />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="saveOutfit">{{ editingOutfit ? $t('common.update') : $t('common.add') }}</el-button>
                <el-button v-if="editingOutfit" @click="editingOutfit = null; newOutfit = {name:'', prompt:''}">{{ $t('common.cancel') }}</el-button>
              </el-form-item>
            </el-form>
          </div>
          
          <!-- Right side: List -->
          <div class="outfits-list-container">
            <div class="outfits-list-header">
              <h4>Outfits ({{ currentCharacterForOutfits?.outfits?.length || 0 }})</h4>
            </div>
            
            <el-empty v-if="!currentCharacterForOutfits?.outfits?.length" description="No outfits added yet" :image-size="60" />
            
            <div class="outfit-items-grid">
              <div v-for="outfit in currentCharacterForOutfits?.outfits" :key="outfit.id" class="outfit-list-item-premium">
                <div class="outfit-item-image">
                  <el-image
                    v-if="outfit.local_path || outfit.image_url"
                    :src="getImageUrl(outfit)"
                    :alt="outfit.name"
                    fit="cover"
                    :preview-src-list="[getImageUrl(outfit)]"
                    hide-on-click-modal
                    preview-teleported
                  />
                  <div v-else class="outfit-image-placeholder">
                    <el-icon :size="24"><Box /></el-icon>
                  </div>
                </div>
                
                <div class="outfit-item-details">
                  <div class="outfit-item-header">
                    <span class="outfit-name">{{ outfit.name }}</span>
                    <el-tag v-if="outfitUsage[outfit.id]" type="info" size="small" effect="plain" class="usage-tag">
                      {{ outfitUsage[outfit.id].length }} shots
                    </el-tag>
                  </div>
                  
                  <div class="outfit-item-prompt">
                    {{ outfit.prompt }}
                  </div>
                  
                  <div v-if="outfit.appearances" class="outfit-item-usage" style="margin-top: 4px; display: flex; align-items: center; flex-wrap: wrap; gap: 4px;">
                    <el-icon><CollectionTag /></el-icon>
                    <span style="font-size: 12px; color: var(--el-text-color-secondary);">Episodes:</span>
                    <span 
                      v-for="node in outfit.appearances.split(',')" 
                      :key="node" 
                      style="font-size: 9px; padding: 1px 5px; background: rgba(64, 158, 255, 0.15); border: 1px solid rgba(64, 158, 255, 0.3); border-radius: 4px; color: #409eff; font-weight: bold;"
                    >
                      {{ node }}
                    </span>
                  </div>

                  <div v-if="outfitUsage[outfit.id]" class="outfit-item-usage">
                    <el-icon><Monitor /></el-icon>
                    <span v-for="(usage, idx) in outfitUsage[outfit.id].slice(0, 2)" :key="idx">
                      Ep {{ usage.episode }}: Shot {{ usage.shot }}{{ idx < 1 && idx < outfitUsage[outfit.id].length - 1 ? ', ' : '' }}
                    </span>
                    <span v-if="outfitUsage[outfit.id].length > 2">...</span>
                  </div>

                  <div class="outfit-item-actions-premium">
                    <el-button-group>
                      <el-button size="small" :icon="Edit" @click="editOutfit(outfit)" title="Edit" />
                      <el-button size="small" type="primary" :icon="MagicStick" @click="generateOutfitImage(outfit)" title="Generate Image" />
                      <el-button size="small" :icon="Upload" @click="uploadOutfitImage(outfit)" title="Upload Image" />
                    </el-button-group>
                    <el-button size="small" type="danger" :icon="Delete" @click="deleteOutfit(outfit)" class="delete-btn" title="Delete" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  Document,
  User,
  Picture,
  Plus,
  Box,
  Edit,
  MagicStick,
  Delete,
  Upload,
  Clock,
  Location,
  Lock,
  Loading,
  DocumentChecked,
  CollectionTag,
} from "@element-plus/icons-vue";
import { dramaAPI } from "@/api/drama";
import { generationAPI } from "@/api/generation";
import { characterLibraryAPI } from "@/api/character-library";
import { propAPI } from "@/api/prop";
import type { Drama } from "@/types/drama";
import {
  AppHeader,
  StatCard,
  EmptyState,
  ImagePreview,
} from "@/components/common";
import { getImageUrl, hasImage } from "@/utils/image";
import { getApiErrorMessage } from "@/utils/request";
import { buildNarrativeMermaidSource } from "@/utils/narrativeGraph";
import NarrativeStoryGraph from "@/components/drama/NarrativeStoryGraph.vue";
import {
  runFullEpisodePipeline,
  getPipelineModelsWithFallback,
  type PipelineStep,
} from "@/composables/useEpisodeFullProduction";

const router = useRouter();
const route = useRoute();

const drama = ref<Drama>();
const activeTab = ref((route.query.tab as string) || "overview");
const scenes = ref<any[]>([]);

let pollingTimer: any = null; // Add polling timer definition

const addCharacterDialogVisible = ref(false);
const addSceneDialogVisible = ref(false);
const addPropDialogVisible = ref(false);
const extractPropsDialogVisible = ref(false);
const extractCharactersDialogVisible = ref(false);
const extractScenesDialogVisible = ref(false);

const editingCharacter = ref<any>(null);
const editingScene = ref<any>(null);
const editingProp = ref<any>(null);
const selectedExtractEpisodeId = ref<number | null>(null);

const narrativeIdea = ref("");
const savingNarrativeIdea = ref(false);
const narrativeLoading = ref(false);
const currentAgentTask = ref(0);
const batchGeneratingCharacterImages = ref(false);
const isIdeaEditing = ref(false);

const handleEditIdea = () => {
  if (episodesCount.value > 0) {
    ElMessageBox.confirm(
      "Cảnh báo: Việc thay đổi Ý tưởng sau khi đã tạo cây cốt truyện có thể làm lệch hướng kịch bản khi chạy lại các Agent. Bạn có chắc chắn muốn chỉnh sửa không?",
      "Cảnh báo thay đổi ý tưởng",
      {
        confirmButtonText: "Đồng ý",
        cancelButtonText: "Hủy",
        type: "warning",
      }
    ).then(() => {
      isIdeaEditing.value = true;
    }).catch(() => {});
  } else {
    isIdeaEditing.value = true;
  }
};

const outfitsDialogVisible = ref(false);
const currentCharacterForOutfits = ref<any>(null);
const newOutfit = ref({ name: '', prompt: '' });
const editingOutfit = ref<any>(null);

const episodeChoiceCount = (row: any): number => {
  const c = row?.choices;
  if (!c) return 0;
  if (Array.isArray(c)) return c.length;
  if (typeof c === "string") {
    try {
      const p = JSON.parse(c);
      return Array.isArray(p) ? p.length : 0;
    } catch {
      return 0;
    }
  }
  return 0;
};

const runAgent = async (agentNumber: number) => {
  const id = route.params.id as string;
  const label = agentNumber === 0 ? "Full pipeline" : `Agent ${agentNumber}`;
  narrativeLoading.value = true;
  currentAgentTask.value = agentNumber;
  try {
    const payload: { user_idea: string; agent_step?: number } = {
      user_idea: narrativeIdea.value.trim(),
    };
    if (agentNumber > 0) {
      payload.agent_step = agentNumber;
    }
    const res = await dramaAPI.generateNarrativeEpisodes(id, payload);
    
    if (res.task_id) {
      ElMessage.info(`Running ${label}...`);
      
      const checkTask = async () => {
        try {
          const task = await generationAPI.getTaskStatus(res.task_id);
          if (task.status === 'completed') {
            await loadDramaData();
            ElMessage.success(`${label} completed!`);
            narrativeLoading.value = false;
            currentAgentTask.value = 0;
          } else if (task.status === 'failed') {
            ElMessage.error(task.error || 'Generation task failed');
            narrativeLoading.value = false;
            currentAgentTask.value = 0;
          } else {
            if (task.progress !== undefined) {
              console.log(`Agent progress: ${task.progress}% - ${task.message}`);
            }
            setTimeout(checkTask, 2000);
          }
        } catch (e) {
          ElMessage.error("Failed to check task status");
          narrativeLoading.value = false;
          currentAgentTask.value = 0;
        }
      };
      
      setTimeout(checkTask, 2000);
    } else {
      await loadDramaData();
      ElMessage.success(`${label} completed!`);
      narrativeLoading.value = false;
      currentAgentTask.value = 0;
    }
  } catch (e: unknown) {
    ElMessage.error(getApiErrorMessage(e, `${label} failed`));
    narrativeLoading.value = false;
    currentAgentTask.value = 0;
  }
};

const saveNarrativeIdea = async () => {
  const id = route.params.id as string;
  savingNarrativeIdea.value = true;
  try {
    await dramaAPI.update(id, { narrative_idea: narrativeIdea.value.trim() });
    if (drama.value) {
      drama.value.narrative_idea = narrativeIdea.value.trim();
    }
    ElMessage.success("Idea saved");
    isIdeaEditing.value = false;
  } catch (e: unknown) {
    ElMessage.error(getApiErrorMessage(e, "Failed to save idea"));
  } finally {
    savingNarrativeIdea.value = false;
  }
};

const newCharacter = ref({
  name: "",
  role: "supporting",
  appearance: "",
  base_image_prompt: "",
  personality: "",
  description: "",
  image_url: "",
  local_path: "",
});

const newProp = ref({
  name: "",
  description: "",
  prompt: "",
  type: "",
  image_url: "",
  local_path: "",
});

const newScene = ref({
  location: "",
  prompt: "",
  image_url: "",
  local_path: "",
});

const episodesCount = computed(() => drama.value?.episodes?.length || 0);
const charactersCount = computed(() => drama.value?.characters?.length || 0);
const scenesCount = computed(() => scenes.value.length);
const charactersNeedingImages = computed(() =>
  (drama.value?.characters || []).filter((c: any) => !c.image_url && !c.local_path),
);
const propsCount = computed(() => drama.value?.props?.length || 0);

const cardAspectRatio = computed(() => {
  if (drama.value?.aspect_ratio === "9:16") {
    return "9/16";
  }
  return "3/4";
});

const sortedEpisodes = computed(() => {
  if (!drama.value?.episodes) return [];
  return [...drama.value.episodes].sort(
    (a, b) => a.episode_number - b.episode_number,
  );
});

const outfitUsage = computed(() => {
  const usageMap: Record<number, { episode: string, shot: number }[]> = {};
  
  if (!drama.value?.episodes) return usageMap;

  drama.value.episodes.forEach((ep: any) => {
    ep.storyboards?.forEach((sb: any) => {
      // Check characters in storyboard for outfits
      // Note: We need to check the storyboard_characters pivot data
      // If characters are returned as objects, check if they have pivot info
      sb.characters?.forEach((char: any) => {
        if (char.pivot?.outfit_id) {
          const oid = char.pivot.outfit_id;
          if (!usageMap[oid]) usageMap[oid] = [];
          usageMap[oid].push({
            episode: ep.title,
            shot: sb.storyboard_number
          });
        }
      });
    });
  });

  return usageMap;
});

const sortedScenes = computed(() => {
  return [...scenes.value].sort((a, b) => {
    // Sort by location name
    const locA = a.location || "";
    const locB = b.location || "";
    return locA.localeCompare(locB);
  });
});


const narrativeMermaidSource = computed(() =>
  buildNarrativeMermaidSource(sortedEpisodes.value as any),
);

const episodeTableRef = ref<any>(null);
const selectedEpisodesRows = ref<any[]>([]);
const episodeSelectAll = ref(false);

const onEpisodeSelectionChange = (rows: any[]) => {
  selectedEpisodesRows.value = rows;
  const total = sortedEpisodes.value.length;
  episodeSelectAll.value = total > 0 && rows.length === total;
};

const onEpisodeSelectAllChange = (val: boolean | string | number) => {
  const t = episodeTableRef.value;
  if (!t) return;
  const on = val === true;
  if (on) {
    sortedEpisodes.value.forEach((row: any) => {
      t.toggleRowSelection(row, true);
    });
  } else {
    t.clearSelection();
  }
};

const autoPipelineDialogVisible = ref(false);
const autoPipelineRunning = ref(false);
const autoPipelineLogLines = ref<string[]>([]);
const autoPipelineQueueStatus = ref<
  "idle" | "running" | "completed" | "failed" | "cancelled"
>("idle");
const autoPipelineProgressPct = ref(0);
const autoPipelineLastFailure = ref<{
  episodeNumber?: number;
  failedStep?: PipelineStep;
  message?: string;
} | null>(null);
let autoPipelineAbort: AbortController | null = null;

const autoPipelineStatusLine = computed(() => {
  switch (autoPipelineQueueStatus.value) {
    case "running":
      return "Running…";
    case "completed":
      return "All selected episodes finished.";
    case "failed":
      return `Stopped: ${autoPipelineLastFailure.value?.message || "error"}`;
    case "cancelled":
      return "Stopped by user.";
    default:
      return "";
  }
});

const cancelAutoPipeline = () => {
  autoPipelineAbort?.abort();
};

const startAutoPipeline = async () => {
  const dramaId = route.params.id as string;
  const snapshot = [...selectedEpisodesRows.value].sort(
    (a, b) => a.episode_number - b.episode_number,
  );
  if (!snapshot.length) return;
  const models = await getPipelineModelsWithFallback(dramaId);
  autoPipelineLogLines.value = [];
  autoPipelineLastFailure.value = null;
  autoPipelineQueueStatus.value = "running";
  autoPipelineRunning.value = true;
  autoPipelineDialogVisible.value = true;
  autoPipelineProgressPct.value = 5;
  autoPipelineAbort = new AbortController();
  const total = snapshot.length;
  let stopped = false;

  for (let i = 0; i < snapshot.length; i++) {
    const ep = snapshot[i];
    const fresh = await dramaAPI.get(dramaId);
    const stillThere = fresh.episodes?.some(
      (e: any) => String(e.id) === String(ep.id),
    );
    if (!stillThere) {
      autoPipelineLogLines.value.push(
        `Skip episode ${ep.episode_number} — removed from project.`,
      );
      continue;
    }

    autoPipelineLogLines.value.push(
      `--- Episode ${ep.episode_number} (id ${ep.id}) ---`,
    );
    const res = await runFullEpisodePipeline({
      dramaId,
      episodeId: String(ep.id),
      episodeNumber: ep.episode_number,
      models,
      signal: autoPipelineAbort.signal,
      onStep: (e) => {
        const detail =
          e.message && e.message.toLowerCase() !== e.status
            ? e.message
            : e.status;
        autoPipelineLogLines.value.push(`[${e.step}] ${detail}`);
      },
    });

    autoPipelineProgressPct.value = Math.min(
      99,
      Math.round(((i + 1) / total) * 100),
    );

    if (!res.ok) {
      stopped = true;
      autoPipelineLastFailure.value = {
        episodeNumber: res.episodeNumber,
        failedStep: res.failedStep,
        message: res.message,
      };
      autoPipelineLogLines.value.push(`FAILED: ${res.message || "unknown"}`);
      autoPipelineQueueStatus.value =
        res.message === "Cancelled" ? "cancelled" : "failed";
      break;
    }
    autoPipelineLogLines.value.push(`Episode ${ep.episode_number} OK`);
    await loadDramaData();
  }

  if (!stopped) {
    autoPipelineQueueStatus.value = "completed";
    autoPipelineProgressPct.value = 100;
  }
  autoPipelineRunning.value = false;
  autoPipelineAbort = null;
  await loadDramaData();
};

const openFailedEpisodeWorkflow = () => {
  const f = autoPipelineLastFailure.value;
  if (f?.episodeNumber == null) return;
  router.push({
    name: "EpisodeWorkflowNew",
    params: {
      id: route.params.id,
      episodeNumber: f.episodeNumber,
    },
    query: f.failedStep ? { focusStep: f.failedStep } : {},
  });
};

// Helper for polling
const startPolling = (
  callback: () => Promise<void>,
  maxAttempts = 20,
  interval = 3000,
) => {
  if (pollingTimer) clearInterval(pollingTimer);

  let attempts = 0;
  pollingTimer = setInterval(async () => {
    attempts++;
    await callback();
    if (attempts >= maxAttempts) {
      if (pollingTimer) clearInterval(pollingTimer);
      pollingTimer = null;
    }
  }, interval);
};

onUnmounted(() => {
  if (pollingTimer) clearInterval(pollingTimer);
});

const loadDramaData = async () => {
  try {
    const data = await dramaAPI.get(route.params.id as string);
    drama.value = data;
    narrativeIdea.value = data.narrative_idea || narrativeIdea.value;
    if (!narrativeIdea.value.trim()) {
      isIdeaEditing.value = true;
    } else {
      isIdeaEditing.value = false;
    }
    loadScenes();
  } catch (error: any) {
    ElMessage.error(error.message || "Failed to load project data");
  }
};

const loadScenes = async () => {
  // 场景数据已经在drama中加载了（后端Preload了Scenes）
  if (drama.value?.scenes) {
    scenes.value = drama.value.scenes;
  } else {
    scenes.value = [];
  }
};

const getStatusType = (status?: string) => {
  const map: Record<string, any> = {
    draft: "info",
    in_progress: "warning",
    completed: "success",
  };
  return map[status || "draft"] || "info";
};

const getStatusText = (status?: string) => {
  const map: Record<string, string> = {
    draft: "Draft",
    in_progress: "In Progress",
    completed: "Completed",
  };
  return map[status || "draft"] || "Draft";
};

const getEpisodeStatusType = (episode: any) => {
  if (episode.storyboards && episode.storyboards.length > 0) return "success";
  if (episode.script_content) return "success";
  if (episode.description || episode.state_snapshot || (episode.scenes && episode.scenes.length > 0)) return "warning";
  if (episode.narrative_node_id) return "info";
  return "info";
};

const getEpisodeStatusText = (episode: any) => {
  if (episode.storyboards && episode.storyboards.length > 0) return "Split";
  if (episode.script_content) return "Script Ready";
  if (episode.description || episode.state_snapshot || (episode.scenes && episode.scenes.length > 0)) return "Built";
  if (episode.narrative_node_id) return "Skeleton";
  return "Draft";
};

const formatDate = (date?: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("en-US");
};

const createNewEpisode = () => {
  const nextEpisodeNumber = episodesCount.value + 1;
  router.push({
    name: "EpisodeWorkflowNew",
    params: {
      id: route.params.id,
      episodeNumber: nextEpisodeNumber,
    },
  });
};

const enterEpisodeWorkflow = (episode: any) => {
  router.push({
    name: "EpisodeWorkflowNew",
    params: {
      id: route.params.id,
      episodeNumber: episode.episode_number,
    },
  });
};

const deleteEpisode = async (episode: any) => {
  try {
    await ElMessageBox.confirm(
      `Delete Episode ${episode.episode_number}? This will also delete all related data (characters, scenes, storyboards, etc.).`,
      "Confirm Deletion",
      {
        confirmButtonText: "Delete",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    // 过滤掉要删除的章节
    const existingEpisodes = drama.value?.episodes || [];
    const updatedEpisodes = existingEpisodes
      .filter((ep) => ep.episode_number !== episode.episode_number)
      .map((ep) => ({
        episode_number: ep.episode_number,
        title: ep.title,
        script_content: ep.script_content,
        description: ep.description,
        duration: ep.duration,
        status: ep.status,
      }));

    // 保存更新后的章节列表
    await dramaAPI.saveEpisodes(drama.value!.id, updatedEpisodes);

    ElMessage.success(`Episode ${episode.episode_number} deleted`);
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const openAddCharacterDialog = () => {
  editingCharacter.value = null;
  newCharacter.value = {
    name: "",
    role: "supporting",
    appearance: "",
    base_image_prompt: "",
    personality: "",
    description: "",
    image_url: "",
    local_path: "",
  };
  addCharacterDialogVisible.value = true;
};

const handleCharacterAvatarSuccess = (response: any) => {
  if (response.data && response.data.url) {
    newCharacter.value.image_url = response.data.url;
    newCharacter.value.local_path = response.data.local_path || "";
  }
};

const handleSceneImageSuccess = (response: any) => {
  if (response.data && response.data.url) {
    newScene.value.image_url = response.data.url;
    newScene.value.local_path = response.data.local_path || "";
  }
};

const beforeAvatarUpload = (file: any) => {
  const isImage = file.type.startsWith("image/");
  const isLt10M = file.size / 1024 / 1024 < 10;

  if (!isImage) {
    ElMessage.error("Only image files are allowed");
  }
  if (!isLt10M) {
    ElMessage.error("Image size cannot exceed 10MB");
  }
  return isImage && isLt10M;
};

const generateCharacterImage = async (character: any) => {
  try {
    await characterLibraryAPI.generateCharacterImage(character.id);
    ElMessage.success("Image generation task submitted");
    startPolling(loadDramaData);
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  }
};

const batchGenerateCharacterImages = async () => {
  const ids = charactersNeedingImages.value.slice(0, 10).map((c: any) => String(c.id));
  if (ids.length === 0) return;
  batchGeneratingCharacterImages.value = true;
  try {
    await characterLibraryAPI.batchGenerateCharacterImages(ids);
    ElMessage.success(`Submitted ${ids.length} character image tasks`);
    startPolling(loadDramaData);
  } catch (error: any) {
    ElMessage.error(error.message || "Batch generation failed");
  } finally {
    batchGeneratingCharacterImages.value = false;
  }
};

const openExtractCharacterDialog = () => {
  extractCharactersDialogVisible.value = true;
  if (sortedEpisodes.value.length > 0 && !selectedExtractEpisodeId.value) {
    selectedExtractEpisodeId.value = sortedEpisodes.value[0].id;
  }
};

const handleExtractCharacters = async () => {
  if (!selectedExtractEpisodeId.value) return;

  try {
    const res = await characterLibraryAPI.extractFromEpisode(
      selectedExtractEpisodeId.value,
    );
    extractCharactersDialogVisible.value = false;

    // 自动刷新几次
    let checkCount = 0;
    const checkInterval = setInterval(() => {
      loadDramaData();
      checkCount++;
      if (checkCount > 10) clearInterval(checkInterval);
    }, 5000);
  } catch (error: any) {
    ElMessage.error(error.message || "Extraction failed");
  }
};

const generateSceneImage = async (scene: any) => {
  try {
    await dramaAPI.generateSceneImage({ scene_id: scene.id });
    ElMessage.success("Image generation task submitted");
    startPolling(loadScenes);
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  }
};

const openExtractSceneDialog = () => {
  extractScenesDialogVisible.value = true;
  if (sortedEpisodes.value.length > 0 && !selectedExtractEpisodeId.value) {
    selectedExtractEpisodeId.value = sortedEpisodes.value[0].id;
  }
};

const openOutfitsDialog = (character: any) => {
  currentCharacterForOutfits.value = character;
  outfitsDialogVisible.value = true;
  editingOutfit.value = null;
  newOutfit.value = { name: '', prompt: '' };
};

const saveOutfit = async () => {
  if (!newOutfit.value.name.trim()) {
    ElMessage.warning('Please enter outfit name');
    return;
  }
  try {
    if (editingOutfit.value) {
      await characterLibraryAPI.updateOutfit(currentCharacterForOutfits.value.id, editingOutfit.value.id, newOutfit.value);
    } else {
      await characterLibraryAPI.createOutfit(currentCharacterForOutfits.value.id, newOutfit.value);
    }
    ElMessage.success('Outfit saved');
    await loadDramaData();
    // Update currentCharacterForOutfits ref from the updated drama data
    const updatedChar = drama.value?.characters?.find((c: any) => c.id === currentCharacterForOutfits.value.id);
    if (updatedChar) currentCharacterForOutfits.value = updatedChar;
    
    editingOutfit.value = null;
    newOutfit.value = { name: '', prompt: '' };
  } catch (err: any) {
    ElMessage.error(err.message || 'Operation failed');
  }
};

const editOutfit = (outfit: any) => {
  editingOutfit.value = outfit;
  newOutfit.value = { name: outfit.name, prompt: outfit.prompt || '' };
};

const deleteOutfit = async (outfit: any) => {
  const usage = outfitUsage.value[outfit.id];
  if (usage && usage.length > 0) {
    try {
      await ElMessageBox.confirm(
        `This outfit is used in ${usage.length} shots across ${[...new Set(usage.map(u => u.episode))].length} episodes. Deleting it will affect those shots. Continue?`,
        'Outfit in Use',
        { type: 'warning', confirmButtonText: 'Delete Anyway', cancelButtonText: 'Keep Outfit' }
      );
    } catch (e) {
      return; // Cancelled
    }
  } else {
    try {
      await ElMessageBox.confirm('Are you sure you want to delete this outfit?', 'Confirm Deletion', { type: 'warning' });
    } catch (e) {
      return; // Cancelled
    }
  }

  try {
    await characterLibraryAPI.deleteOutfit(currentCharacterForOutfits.value.id, outfit.id);
    ElMessage.success('Outfit deleted');
    await loadDramaData();
    const updatedChar = drama.value?.characters?.find((c: any) => c.id === currentCharacterForOutfits.value.id);
    if (updatedChar) currentCharacterForOutfits.value = updatedChar;
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to delete outfit');
  }
};

const uploadOutfitImage = (outfit: any) => {
  // Reuse currentUploadTarget logic
  currentUploadTarget.value = { id: outfit.id, type: 'outfit', parentId: currentCharacterForOutfits.value.id };
  uploadDialogVisible.value = true;
};


const generateOutfitImage = async (outfit: any) => {
  try {
    await characterLibraryAPI.generateOutfitImage(currentCharacterForOutfits.value.id, outfit.id, {});
    ElMessage.success('Outfit image generation started');
    startPolling(async () => {
      await loadDramaData();
      const updatedChar = drama.value?.characters?.find((c: any) => c.id === currentCharacterForOutfits.value.id);
      if (updatedChar) currentCharacterForOutfits.value = updatedChar;
    });
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed');
  }
};

const handleExtractScenes = async () => {
  if (!selectedExtractEpisodeId.value) return;

  try {
    const res = await dramaAPI.extractBackgrounds(
      selectedExtractEpisodeId.value.toString(),
    );
    extractScenesDialogVisible.value = false;

    // 自动刷新几次
    let checkCount = 0;
    const checkInterval = setInterval(() => {
      loadScenes();
      checkCount++;
      if (checkCount > 10) clearInterval(checkInterval);
    }, 5000);
  } catch (error: any) {
    ElMessage.error(error.message || "Extraction failed");
  }
};

const saveCharacter = async () => {
  if (!newCharacter.value.name.trim()) {
    ElMessage.warning("Please enter character name");
    return;
  }

  try {
    if (editingCharacter.value) {
      // Edit existing character using dedicated update endpoint
      await dramaAPI.updateCharacter(editingCharacter.value.id, {
        name: newCharacter.value.name,
        role: newCharacter.value.role,
        appearance: newCharacter.value.appearance,
        base_image_prompt: newCharacter.value.base_image_prompt,
        personality: newCharacter.value.personality,
        description: newCharacter.value.description,
        image_url: newCharacter.value.image_url,
        local_path: newCharacter.value.local_path,
      });
      ElMessage.success("Character updated");
    } else {
      // Add new character
      const allCharacters = [
        ...(drama.value?.characters || []).map((c) => ({
          name: c.name,
          role: c.role,
          appearance: c.appearance,
          base_image_prompt: c.base_image_prompt,
          personality: c.personality,
          description: c.description,
          image_url: c.image_url,
          local_path: c.local_path,
        })),
        newCharacter.value,
      ];

      await dramaAPI.saveCharacters(drama.value!.id, allCharacters);
      ElMessage.success("Character added");
    }

    addCharacterDialogVisible.value = false;
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "Operation failed");
  }
};

const editCharacter = (character: any) => {
  editingCharacter.value = character;
  newCharacter.value = {
    name: character.name,
    role: character.role || "supporting",
    appearance: character.appearance || "",
    base_image_prompt: character.base_image_prompt || "",
    personality: character.personality || "",
    description: character.description || "",
    image_url: character.image_url || "",
    local_path: character.local_path || "",
  };
  addCharacterDialogVisible.value = true;
};

const deleteCharacter = async (character: any) => {
  if (!character.id) {
    ElMessage.error("Character ID does not exist");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Delete character "${character.name}"? This cannot be undone.`,
      "Confirm Deletion",
      {
        confirmButtonText: "Delete",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    await characterLibraryAPI.deleteCharacter(character.id);
    ElMessage.success("Character deleted");
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      console.error("Failed to delete character:", error);
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const openAddSceneDialog = () => {
  editingScene.value = null;
  newScene.value = {
    location: "",
    prompt: "",
    image_url: "",
  };
  addSceneDialogVisible.value = true;
};

const saveScene = async () => {
  if (!newScene.value.location.trim()) {
    ElMessage.warning("Please enter scene name");
    return;
  }

  try {
    if (editingScene.value) {
      // Update existing scene
      await dramaAPI.updateScene(editingScene.value.id, {
        location: newScene.value.location,
        description: newScene.value.prompt,
        image_url: newScene.value.image_url,
        local_path: newScene.value.local_path,
      });
      // prompt field in Update is description or prompt? Check backend.
      // UpdateSceneRequest has Description *string.
      // And also ImagePrompt *string and VideoPrompt *string.
      // The backend model has Prompt string.
      // Checking backend handler:
      /*
        if req.Description != nil { updates["description"] = req.Description }
        if req.ImagePrompt != nil { updates["image_prompt"] = req.ImagePrompt }
      */
      // But CreateScene uses Prompt.
      // Let's assume description maps to Prompt or Description.
      // Wait, UpdateSceneRequest has Description but NO Prompt field?
      // Let's check backend UpdateSceneRequest struct again.
      // It has `ImagePrompt` and `VideoPrompt`, and `Description`.
      // But `Prompt` usually refers to image prompt in Scene model?
      // `models.Scene` has `Prompt` string.
      // `CreateScene` sets `Prompt: req.Prompt`.
      // `UpdateScene` handler:
      /*
      	if req.Description != nil {
      		updates["description"] = req.Description
      	}
      */
      // It seems UpdateScene doesn't support updating the main `Prompt` field directly via UpdateSceneRequest?
      // Wait, `UpdateScenePrompt` endpoint exists! `/scenes/:id/prompt`
      // But we probably want to update everything in one go.
      // I should update UpdateSceneRequest in backend if needed or use UpdateScenePrompt separately.
      // For now, let's look at scene model:
      // Scene struct: Location, Time, Description, Prompt...
      // Let's use `description` for now as it's available in Update.
      // Or if `prompt` is critical, I might need to call UpdateScenePrompt too.
      // Let's check `CreateScene` again. It uses `Prompt`.

      // Let's just update prompt via specific endpoint if needed, or mapping description to description.
      // Actually `newScene.prompt` is mapped to `description` in my current code for Update.
      // Let's stick with that for now or fix backend to support prompt update in general update.
    } else {
      // Create new scene
      await dramaAPI.createScene({
        drama_id: drama.value!.id,
        location: newScene.value.location,
        prompt: newScene.value.prompt,
        description: newScene.value.prompt,
        image_url: newScene.value.image_url,
        local_path: newScene.value.local_path,
      });
    }

    ElMessage.success(editingScene.value ? "Scene updated" : "Scene added");
    addSceneDialogVisible.value = false;
    await loadScenes();
  } catch (error: any) {
    ElMessage.error(error.message || "Operation failed");
  }
};

const editScene = (scene: any) => {
  editingScene.value = scene;
  newScene.value = {
    location: scene.location || scene.name || "",
    prompt: scene.prompt || scene.description || "",
    image_url: scene.image_url || "",
    local_path: scene.local_path || "",
  };
  addSceneDialogVisible.value = true;
};

const deleteScene = async (scene: any) => {
  if (!scene.id) {
    ElMessage.error("Scene ID does not exist");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Delete scene "${scene.name || scene.location}"? This cannot be undone.`,
      "Confirm Deletion",
      {
        confirmButtonText: "Delete",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    await dramaAPI.deleteScene(scene.id.toString());
    ElMessage.success("Scene deleted");
    await loadScenes();
  } catch (error: any) {
    if (error !== "cancel") {
      console.error("Failed to delete scene:", error);
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const openAddPropDialog = () => {
  editingProp.value = null;
  newProp.value = {
    name: "",
    description: "",
    prompt: "",
    type: "",
    image_url: "",
    local_path: "",
  };
  addPropDialogVisible.value = true;
};

const saveProp = async () => {
  if (!newProp.value.name.trim()) {
    ElMessage.warning("Please enter prop name");
    return;
  }

  try {
    const propData = {
      drama_id: drama.value!.id,
      name: newProp.value.name,
      description: newProp.value.description,
      prompt: newProp.value.prompt,
      type: newProp.value.type,
      image_url: newProp.value.image_url,
      local_path: newProp.value.local_path,
    };

    if (editingProp.value) {
      await propAPI.update(editingProp.value.id, propData);
      ElMessage.success("Prop updated");
    } else {
      await propAPI.create(propData as any);
      ElMessage.success("Prop added");
    }

    addPropDialogVisible.value = false;
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "Operation failed");
  }
};

const editProp = (prop: any) => {
  editingProp.value = prop;
  newProp.value = {
    name: prop.name,
    description: prop.description || "",
    prompt: prop.prompt || "",
    type: prop.type || "",
    image_url: prop.image_url || "",
    local_path: prop.local_path || "",
  };
  addPropDialogVisible.value = true;
};

const deleteProp = async (prop: any) => {
  try {
    await ElMessageBox.confirm(
      `Delete prop "${prop.name}"? This cannot be undone.`,
      "Confirm Deletion",
      {
        confirmButtonText: "Delete",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    await propAPI.delete(prop.id);
    ElMessage.success("Prop deleted");
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const generatePropImage = async (prop: any) => {
  if (!prop.prompt) {
    ElMessage.warning("Please set prop image prompt first");
    editProp(prop);
    return;
  }

  try {
    await propAPI.generateImage(prop.id);
    ElMessage.success("Image generation task submitted");
    startPolling(loadDramaData);
  } catch (error: any) {
    ElMessage.error(error.message || "Generation failed");
  }
};

const handlePropImageSuccess = (response: any) => {
  if (response.data && response.data.url) {
    newProp.value.image_url = response.data.url;
    newProp.value.local_path = response.data.local_path || "";
  }
};

const openExtractDialog = () => {
  extractPropsDialogVisible.value = true;
  if (sortedEpisodes.value.length > 0 && !selectedExtractEpisodeId.value) {
    selectedExtractEpisodeId.value = sortedEpisodes.value[0].id;
  }
};

const handleExtractProps = async () => {
  if (!selectedExtractEpisodeId.value) return;

  try {
    const res = await propAPI.extractFromScript(selectedExtractEpisodeId.value);
    extractPropsDialogVisible.value = false;

    // 自动刷新几次
    let checkCount = 0;
    const checkInterval = setInterval(() => {
      loadDramaData();
      checkCount++;
      if (checkCount > 10) clearInterval(checkInterval);
    }, 5000);
  } catch (error: any) {
    ElMessage.error(error.message || t("common.failed"));
  }
};

onMounted(() => {
  loadDramaData();
  loadScenes();

  // 如果有query参数指定tab，切换到对应tab
  if (route.query.tab) {
    activeTab.value = route.query.tab as string;
  }
});
</script>

<style scoped>
/* ========================================
   Page Layout / 页面布局 - 紧凑边距
   ======================================== */
.page-container {
  min-height: 100vh;
  background: var(--bg-primary);
  /* padding: var(--space-2) var(--space-3); */
  transition: background var(--transition-normal);
}

@media (min-width: 768px) {
  .page-container {
    /* padding: var(--space-3) var(--space-4); */
  }
}

@media (min-width: 1024px) {
  .page-container {
    /* padding: var(--space-4) var(--space-5); */
  }
}

.content-wrapper {
  margin: 0 auto;
  width: 100%;
}

/* ========================================
   Stats Grid / 统计网格 - 紧凑间距
   ======================================== */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(1, 1fr);
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

@media (min-width: 640px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-3);
  }
}

@media (min-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* ========================================
   Tabs Wrapper / 标签页容器 - 紧凑内边距
   ======================================== */
.tabs-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-3);
  box-shadow: var(--shadow-card);
}

@media (min-width: 768px) {
  .tabs-wrapper {
    padding: var(--space-4);
  }
}

/* ========================================
   Tab Header / 标签页头部
   ======================================== */
.tab-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

@media (min-width: 640px) {
  .tab-header {
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
  }
}

.tab-header h2 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

/* ========================================
   Character & Scene Cards / 角色场景卡片
   ======================================== */
.character-card,
.scene-card {
  margin-bottom: var(--space-4);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-xl);
  overflow: hidden;
  transition: all var(--transition-normal);
}

.character-card:hover,
.scene-card:hover {
  border-color: var(--border-secondary);
  box-shadow: var(--shadow-card-hover);
}

.character-card :deep(.el-card__body),
.scene-card :deep(.el-card__body) {
  padding: 0;
}

.scene-preview {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 160px;
  background: linear-gradient(135deg, var(--accent) 0%, #06b6d4 100%);
  overflow: hidden;
}

.scene-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--transition-normal);
}

.scene-card:hover .scene-preview img {
  transform: scale(1.05);
}

.scene-placeholder {
  color: rgba(255, 255, 255, 0.7);
}

.scene-info {
  text-align: center;
  padding: var(--space-4);
}

.scene-info h4 {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.scene-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: center;
  padding: 0 var(--space-4) var(--space-4);
}

.desc {
  font-size: 0.8125rem;
  color: var(--text-muted);
  margin: var(--space-2) 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

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
  padding: var(--space-3);
  pointer-events: none;
}

.overlay-top {
  display: flex;
  justify-content: flex-end;
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
  }
}

.character-content {
  padding: var(--space-3);
  background: var(--bg-card);
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.char-description {
  font-size: 0.875rem;
  color: var(--text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.outfits-section {
  .outfits-header {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: var(--space-2);
  }

  .outfits-scroll {
    display: flex;
    gap: var(--space-2);
    overflow-x: auto;
    padding-bottom: var(--space-1);
    
    &::-webkit-scrollbar {
      height: 4px;
    }
    
    &::-webkit-scrollbar-thumb {
      background: var(--border-primary);
      border-radius: 2px;
    }
  }
}

.outfit-card-mini {
  width: 60px;
  flex-shrink: 0;
  position: relative;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--border-primary);
  
  .outfit-img {
    width: 100%;
    height: 80px;
    display: block;
  }
  
  .outfit-name-tag {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    background: rgba(0, 0, 0, 0.6);
    color: white;
    font-size: 9px;
    padding: 2px;
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.character-actions-premium {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-primary);
  
  .delete-btn {
    color: var(--text-muted);
    &:hover {
      color: var(--danger);
      background: var(--danger-light);
    }
  }
}

.outfits-list-container {
  flex: 1.5;
  max-height: 500px;
  overflow-y: auto;
  padding-left: var(--space-4);
  border-left: 1px solid var(--border-primary);
}

.outfit-list-item-premium {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-3);
  transition: all 0.3s;
  
  &:hover {
    border-color: var(--accent);
    box-shadow: var(--shadow-md);
  }
}

.outfit-item-image {
  width: 80px;
  height: 120px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-card);
  flex-shrink: 0;
  border: 1px solid var(--border-primary);
  
  .el-image {
    width: 100%;
    height: 100%;
  }
}

.outfit-image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.outfit-item-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  overflow: hidden;
}

.outfit-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  .outfit-name {
    font-weight: 600;
    color: var(--text-primary);
  }
}

.outfit-item-prompt {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.outfit-item-usage {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.outfit-item-actions-premium {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: var(--space-2);
}

.character-actions,
.scene-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  padding: 12px;
  flex-wrap: wrap;
  border-top: 1px solid var(--border-primary);
  background: var(--bg-secondary);
}

.character-actions .el-button,
.scene-actions .el-button {
  margin: 0 !important;
}

.image-placeholder-small {
  width: 80px;
  height: 80px;
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--text-muted);
  font-size: 10px;
  gap: 4px;
  border: 1px dashed var(--border-primary);
}

.outfit-image-container {
  flex-shrink: 0;
}

.outfit-item-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}

.empty-icon {
  color: var(--accent);
}

/* ========================================
   Dark Mode / 深色模式
   ======================================== */
.dark .tabs-wrapper {
  background: var(--bg-card);
}

.dark :deep(.el-card) {
  background: var(--bg-card);
  border-color: var(--border-primary);
}

.dark :deep(.el-card__header) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

.dark :deep(.el-table) {
  background: var(--bg-card);
  --el-table-bg-color: var(--bg-card);
  --el-table-tr-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-secondary);
  --el-fill-color-lighter: var(--bg-secondary);
}

.dark :deep(.el-table th),
.dark :deep(.el-table tr) {
  background: var(--bg-card);
}

.dark :deep(.el-table td),
.dark :deep(.el-table th) {
  border-color: var(--border-primary);
}

.dark :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background: var(--bg-secondary);
}

.dark :deep(.el-table__body tr:hover > td) {
  background: var(--bg-card-hover) !important;
}

.dark :deep(.el-descriptions) {
  background: var(--bg-card);
}

.dark :deep(.el-descriptions__label) {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border-color: var(--border-primary);
}

.dark :deep(.el-descriptions__content) {
  background: var(--bg-card);
  color: var(--text-primary);
  border-color: var(--border-primary);
}

.dark :deep(.el-descriptions__cell) {
  border-color: var(--border-primary);
}

/* ========================================
   Project Info Card / 项目信息卡片
   ======================================== */
.project-info-card {
  margin-top: var(--space-5);
  border-radius: var(--radius-lg);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.project-descriptions {
  width: 100%;
}

:deep(.project-descriptions .el-descriptions__label) {
  width: 120px;
  font-weight: 500;
  color: var(--text-secondary);
}

:deep(.project-descriptions .el-descriptions__content) {
  min-width: 150px;
}

.info-value {
  font-weight: 500;
  color: var(--text-primary);
}

.info-desc {
  color: var(--text-secondary);
  line-height: 1.6;
}

.dark :deep(.el-dialog) {
  background: var(--bg-card);
}

.dark :deep(.el-dialog__header) {
  background: var(--bg-card);
}

.dark :deep(.el-form-item__label) {
  color: var(--text-primary);
}

.dark :deep(.el-input__wrapper) {
  background: var(--bg-secondary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

.dark :deep(.el-input__inner) {
  color: var(--text-primary);
}

.dark :deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

.episode-batch-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
}

/* Plot Outline cell – 2-line clamp, readable */
.plot-outline-cell {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 12.5px;
  line-height: 1.55;
  color: var(--el-text-color-regular);
  cursor: default;
  max-height: 3.2em;
}

.plot-outline-empty {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

/* Operations cell – compact, vertically centered */
.episode-ops-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: nowrap;
}

.auto-pipeline-status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-primary, var(--el-border-color));
  background: var(--bg-secondary, var(--el-fill-color-light));
  position: relative;
  z-index: 1;
}

.auto-pipeline-status-text {
  font-size: 13px;
  color: var(--text-primary, var(--el-text-color-primary));
  line-height: 1.4;
}

.auto-pipeline-pct {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-muted, var(--el-text-color-secondary));
  flex-shrink: 0;
}

.auto-pipeline-progress {
  margin-bottom: 12px;
  position: relative;
  z-index: 1;
}

.auto-pipeline-dialog :deep(.el-dialog__body) {
  padding-top: 12px;
}

.auto-pipeline-log {
  max-height: 240px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.45;
  font-family: ui-monospace, monospace;
  background: var(--bg-secondary, var(--el-fill-color-lighter));
  color: var(--text-primary, var(--el-text-color-primary));
  padding: 10px;
  border-radius: 6px;
  margin-top: 0;
  border: 1px solid var(--border-primary, var(--el-border-color-lighter));
  position: relative;
  z-index: 0;
  clear: both;
}

.auto-pipeline-log .log-line {
  margin: 4px 0;
  word-break: break-word;
  white-space: pre-wrap;
}
</style>
