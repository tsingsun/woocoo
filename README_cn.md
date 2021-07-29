# WooCoo

### jwt验证

生成RSA公私钥
```
openssl genrsa -out jwt_private.pem 1024
openssl rsa -in jwt_private.pem -pubout -out rsa_public_key.pem
```

