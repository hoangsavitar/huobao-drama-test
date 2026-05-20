export interface AIServiceConfig {
  id: number
  service_type: AIServiceType
  provider?: string
  name: string
  base_url: string
  api_key: string
  model: string | string[]
  endpoint: string
  query_endpoint?: string
  priority: number
  is_active: boolean
  settings?: string
  created_at: string
  updated_at: string
}

export type AIServiceType = 'text' | 'image' | 'video'

export interface CreateAIConfigRequest {
  service_type: AIServiceType
  provider?: string
  name: string
  base_url: string
  api_key: string
  model: string | string[]
  endpoint?: string
  query_endpoint?: string
  priority?: number
  settings?: string
}

export interface UpdateAIConfigRequest {
  name?: string
  provider?: string
  base_url?: string
  api_key?: string
  model?: string | string[]
  endpoint?: string
  query_endpoint?: string
  priority?: number
  is_active?: boolean
  settings?: string
}

export interface TestConnectionRequest {
  base_url: string
  api_key: string
  model: string | string[]
  provider?: string
  endpoint?: string
  query_endpoint?: string
}

export interface AIServiceProvider {
  id: number
  name: string
  display_name: string
  service_type: AIServiceType
  default_url: string
  description: string
  is_active: boolean
  created_at: string
  updated_at: string
}
