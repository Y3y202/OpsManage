import api from '../index'

export interface ApiResponse<T = any> {
  code: number
  msg: string
  data: T
}

export function unwrap<T = any>(response: ApiResponse<T>): T {
  return response.data
}

export default api
