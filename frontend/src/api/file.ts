import request from './request'

export const listFiles = (path: string) => request.get('/files/list', { params: { path } })
export const readFile = (path: string) => request.get('/files/read', { params: { path } })
export const saveFile = (path: string, content: string) => request.post('/files/save', { path, content })
export const renameFile = (oldPath: string, newPath: string) => request.post('/files/rename', { old_path: oldPath, new_path: newPath })
export const deleteFile = (path: string) => request.delete('/files', { params: { path } })
export const mkdir = (path: string) => request.post('/files/mkdir', { path })
export const copyFile = (src: string, dst: string) => request.post('/files/copy', { src, dst })
