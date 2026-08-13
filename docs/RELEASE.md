# Processo de release

1. Todos os checks da PR devem passar e a mudança deve ser revisada.
2. O proprietário faz o merge manual em `main`.
3. Crie a tag assinada `v3.0.0` em um commit pertencente a `main`.
4. O workflow baixa ferramentas de versões fixas e valida SHA-256.
5. O frontend e `MP3_Downloader.exe` são reconstruídos.
6. O script cria `MP3_Downloader_v3.0.0_Windows.zip` somente com arquivos permitidos.
7. O pacote montado é iniciado e deve informar todas as ferramentas como prontas.
8. O workflow gera `checksums.sha256`, atesta os artefatos e publica a release sem substituir uma existente.

Verificação pelo usuário:

```powershell
Get-FileHash .\MP3_Downloader_v3.0.0_Windows.zip -Algorithm SHA256
gh attestation verify .\MP3_Downloader_v3.0.0_Windows.zip --repo misaeldasilva123ms96-commits/BAIXAR-MP3
```

O pacote não pode incluir logs, MP3, caches, histórico, temporários, configuração pessoal ou secrets. A atestação não substitui assinatura Authenticode; essa limitação deve permanecer documentada.
