#ifndef ATTESTATION_WRAPPER_H
#define ATTESTATION_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    void* client;
    void* logger;
} AttestationContext;

AttestationContext* create_attestation_client();
char* perform_attestation(AttestationContext* ctx, const char* endpoint, const char* nonce);
void destroy_attestation_client(AttestationContext* ctx);

#ifdef __cplusplus
}
#endif

#endif // ATTESTATION_WRAPPER_H
