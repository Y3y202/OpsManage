import api from '../index'

export interface ApiResponse<T = any> {
  code: number
  msg: string
  data: T
}

export function unwrap<T = any>(response: ApiResponse<T> | T): T {
  if (response && typeof response === 'object' && 'code' in response && 'data' in response) {
    return (response as ApiResponse<T>).data
  }
  return response as T
}

export default api
