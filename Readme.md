- Lấy thông tin metric heap mem từ api /jmx của impala, convert json tới các metric cho prometheus
- Biến môi trường:
```
  - PORT: port exporter
  - NODE_IP: danh sách ip của impala node, cách nhau bởi dấu phẩy
  - IMPALA_PORT: port của service impala
```