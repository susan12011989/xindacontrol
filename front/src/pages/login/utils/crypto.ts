// 将 PEM 公钥导入为 CryptoKey（RSA-OAEP SHA-256）
export async function importRsaPublicKey(pem: string): Promise<CryptoKey> {
  const b64 = pem
    .replace("-----BEGIN PUBLIC KEY-----", "")
    .replace("-----END PUBLIC KEY-----", "")
    .replace(/\s+/g, "")
  const der = Uint8Array.from(atob(b64), (c, _i) => c.charCodeAt(0))
  return crypto.subtle.importKey(
    "spki",
    der,
    {
      name: "RSA-OAEP",
      hash: "SHA-256"
    },
    false,
    ["encrypt"]
  )
}

export async function rsaOaepEncrypt(key: CryptoKey, data: BufferSource): Promise<ArrayBuffer> {
  return crypto.subtle.encrypt(
    {
      name: "RSA-OAEP"
    },
    key,
    data
  )
}

export function base64ToArrayBuffer(b64: string): ArrayBuffer {
  const bin = atob(b64)
  const len = bin.length
  const bytes = new Uint8Array(len)
  for (let i = 0; i < len; i++) bytes[i] = bin.charCodeAt(i)
  return bytes.buffer
}

// -------- 降级分支（不在安全上下文时，使用 node-forge 实现 RSA-OAEP 加密） --------
// 注意：需要在 package.json 增加依赖 "node-forge"
export async function rsaOaepEncryptCompat(pem: string, data: Uint8Array): Promise<string> {
  if (typeof window !== "undefined" && window.isSecureContext && window.crypto?.subtle) {
    const key = await importRsaPublicKey(pem)
    // 将视图严格转为 ArrayBuffer，避免 TS 对 ArrayBufferLike/SharedArrayBuffer 的类型报错
    const ab: ArrayBuffer = (data.buffer as ArrayBuffer).slice(
      data.byteOffset,
      data.byteOffset + data.byteLength
    )
    const cipher = await rsaOaepEncrypt(key, ab)
    return btoa(String.fromCharCode(...new Uint8Array(cipher)))
  }
  // 非安全上下文：使用 node-forge（CJS/ESM 兼容处理）
  const forgeMod = await import("node-forge")
  const forge: any = (forgeMod as any).default ?? forgeMod
  const publicKey = forge.pki.publicKeyFromPem(pem)
  // RSA-OAEP with SHA-256 + MGF1(SHA-256)
  const md = forge.md.sha256.create()
  const mgf1 = forge.mgf.mgf1.create(forge.md.sha256.create())
  // 将 Uint8Array 转为二进制字符串
  const binary = String.fromCharCode(...data)
  const encrypted = publicKey.encrypt(binary, "RSA-OAEP", { md, mgf1 })
  return forge.util.encode64(encrypted)
}
