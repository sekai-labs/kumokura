# Provider Compatibility Matrix

Kumokura supports standard S3 APIs across public cloud and on-premise object storage systems.

| Provider | Path-Style Addressing | Object Lock | Bucket Versioning | Multipart Uploads | Storage Classes | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **AWS S3** | Virtual-Hosted (Recommended) | Yes | Yes | Yes | Standard, IA, Glacier, Deep Archive, etc. | Full feature support |
| **MinIO** | Path-Style (Recommended) | Yes | Yes | Yes | STANDARD | High-performance self-hosted S3 |
| **Cloudflare R2** | Virtual-Hosted | No | No | Yes | STANDARD | Zero egress fee object storage |
| **Wasabi** | Virtual-Hosted | Yes | Yes | Yes | STANDARD | Hot cloud storage |
| **Backblaze B2** | Virtual-Hosted | Yes | Yes | Yes | STANDARD | S3 Compatible API |
| **Ceph Object Gateway** | Path-Style | Yes | Yes | Yes | STANDARD | RADOS Gateway S3 API |
| **DigitalOcean Spaces** | Path-Style | No | No | Yes | STANDARD | High-availability droplet storage |
