import request from './request'

export const listFiles = (path: string) => request.get('/api/files/list', { params: { path } })
export const readFile = (path: string) => request.get('/api/files/read', { params: { path } })
export const saveFile = (path: string, content: string) => request.post('/api/files/save', { path, content })
export const renameFile = (oldPath: string, newPath: string) => request.post('/api/files/rename', { old_path: oldPath, new_path: newPath })
export const deleteFile = (path: string) => request.delete('/api/files', { params: { path } })
export const mkdir = (path: string) => request.post('/api/files/mkdir', { path })
export const copyFile = (src: string, dst: string) => request.post('/api/files/copy', { src, dst })
