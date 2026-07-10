// Entry detail with on-device verification: the phone recomputes the
// canonical hash and Merkle inclusion proof itself (expo-crypto SHA-256),
// trusting nothing but the anchored root — the same check as the web and the
// standalone CLI verifier.
import { useQuery } from "@tanstack/react-query";
import * as Crypto from "expo-crypto";
import { Stack, useLocalSearchParams } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

import { publicApi } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import { verifyProof, type VerificationResult } from "@/lib/verify";
import { Body, Card, Mono, Muted, Row, StatusBadge, Title } from "@/components/ui";
import { radius, spacing, usePalette } from "@/theme";

const sha256Hex = (value: string) =>
  Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, value);

export default function EntryDetail() {
  const palette = usePalette();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [result, setResult] = useState<VerificationResult | null>(null);
  const [checking, setChecking] = useState(false);

  const proof = useQuery({
    queryKey: ["proof", id],
    queryFn: () => publicApi.entryProof(id),
  });

  const entry = proof.data?.entry;

  const runVerification = async () => {
    if (!proof.data) return;
    setChecking(true);
    try {
      setResult(await verifyProof(proof.data, sha256Hex));
    } finally {
      setChecking(false);
    }
  };

  return (
    <ScrollView contentContainerStyle={{ padding: spacing.lg, gap: spacing.md }}>
      <Stack.Screen options={{ title: id }} />

      {proof.isLoading ? (
        <Card>
          <Muted>Loading entry and proof…</Muted>
        </Card>
      ) : proof.isError || !proof.data || !entry ? (
        <Card>
          <Muted>
            No proof is available for this entry yet — it may not be anchored.
          </Muted>
        </Card>
      ) : (
        <>
          <Card style={{ gap: spacing.sm }}>
            <Row>
              <Title>{entry.institutionId}</Title>
              <StatusBadge status={entry.status} />
            </Row>
            <Muted>
              FY{entry.fiscalYear} · {entry.effectiveDate} · {entry.entryType}
            </Muted>
            <View
              style={{
                borderTopWidth: 1,
                borderTopColor: palette.border,
                paddingTop: spacing.sm,
                gap: spacing.xs,
              }}
            >
              {entry.lines.map((line, index) => (
                <Row key={index}>
                  <Muted style={{ flex: 1 }}>
                    {line.accountCode} · {line.direction}
                  </Muted>
                  <Mono>{formatMoney(line.amountMinor, line.currency)}</Mono>
                </Row>
              ))}
            </View>
          </Card>

          <Card style={{ gap: spacing.sm }}>
            <Body style={{ fontWeight: "700" }}>Independent verification</Body>
            <Muted>
              This device recomputes the entry's canonical hash and walks the
              Merkle proof to the anchored root. If anything was modified
              after anchoring, the check fails.
            </Muted>
            <Muted>
              Anchored via {proof.data.anchor.backend} (
              {proof.data.anchor.backendRef})
            </Muted>

            {result ? (
              <View
                style={{
                  borderRadius: radius.control,
                  borderWidth: 1,
                  borderColor: result.verified ? palette.success : palette.danger,
                  padding: spacing.md,
                  gap: spacing.xs,
                }}
              >
                <Body
                  style={{
                    color: result.verified ? palette.success : palette.danger,
                    fontWeight: "700",
                  }}
                >
                  {result.verified
                    ? "✓ VERIFIED — anchored, unmodified"
                    : "✗ FAILED — does not match the anchored root"}
                </Body>
                <Mono style={{ color: palette.textMuted }}>
                  leaf {result.recomputedLeaf}
                </Mono>
                <Mono style={{ color: palette.textMuted }}>
                  root {result.recomputedRoot}
                </Mono>
              </View>
            ) : null}

            <Pressable
              accessibilityRole="button"
              onPress={runVerification}
              disabled={checking}
              style={{
                backgroundColor: palette.primary,
                borderRadius: radius.control,
                padding: spacing.md,
                alignItems: "center",
                opacity: checking ? 0.6 : 1,
              }}
            >
              <Text style={{ color: "#ffffff", fontWeight: "700" }}>
                {checking ? "Verifying…" : result ? "Verify again" : "Verify on this device"}
              </Text>
            </Pressable>
          </Card>
        </>
      )}
    </ScrollView>
  );
}
