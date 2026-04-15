import type { AxiosError, AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import axios from 'axios'
import { ElMessage } from 'element-plus'

interface CustomAxiosInstance extends Omit<AxiosInstance, 'get' | 'post' | 'put' | 'patch' | 'delete'> {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T>
  put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T>
  patch<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T>
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T>
}

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 600000, // 10-minute timeout to match backend AI generation endpoints
  headers: {
    'Content-Type': 'application/json'
  }
}) as CustomAxiosInstance

// Open source version - no auth token required
request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    return config
  },
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.success) {
      return res.data
    } else {
      // Let business code handle errors directly
      return Promise.reject(new Error(res.error?.message || 'Request failed'))
    }
  },
  (error: AxiosError<any>) => {
    // Let callers handle errors based on their specific context
    return Promise.reject(error)
  }
)

/** Backend JSON error shape: { success, error?: { message } } */
export function getApiErrorMessage(err: unknown, fallback = 'Request failed'): string {
  const e = err as AxiosError<{ error?: { message?: string }; message?: string }>
  return e?.response?.data?.error?.message ?? e?.response?.data?.message ?? e?.message ?? fallback
}

export default request
