#include <stdlib.h>
#include <string.h>
#include <AttestationClient.h>
#include "Logger.h"

extern "C" {

typedef struct {
    void* client;
    void* logger;
} AttestationContext;

AttestationContext* create_attestation_client() {
    AttestationContext* ctx = (AttestationContext*)malloc(sizeof(AttestationContext));
    ctx->logger = new Logger();
    ctx->client = nullptr;

    if (!Initialize((Logger*)ctx->logger, (AttestationClient**)&ctx->client)) {
        free(ctx);
        return nullptr;
    }

    return ctx;
}

char* perform_attestation(AttestationContext* ctx, const char* endpoint, const char* nonce) {
    if (ctx == nullptr || ctx->client == nullptr) {
        return nullptr;
    }

    AttestationClient* client = (AttestationClient*)ctx->client;

    attest::ClientParameters params = {};
    params.attestation_endpoint_url = (unsigned char*)endpoint;

    char payload[512];
    snprintf(payload, sizeof(payload), "{\"nonce\":\"%s\"}", nonce);
    params.client_payload = (unsigned char*)payload;
    params.version = CLIENT_PARAMS_VERSION;

    unsigned char* jwt = nullptr;
    attest::AttestationResult result = client->Attest(params, &jwt);

    if (result.code_ != attest::AttestationResult::ErrorCode::SUCCESS) {
        return nullptr;
    }

    char* jwt_copy = strdup((char*)jwt);
    client->Free(jwt);

    return jwt_copy;
}

void destroy_attestation_client(AttestationContext* ctx) {
    if (ctx != nullptr) {
        Uninitialize();
        if (ctx->logger != nullptr) {
            delete (Logger*)ctx->logger;
        }
        free(ctx);
    }
}

} // extern "C"
