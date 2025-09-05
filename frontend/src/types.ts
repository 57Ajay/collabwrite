import { z } from "zod/mini";

const stringToDate = z.codec(
    z.string(),
    z.date(),
    {
        decode: (isoString) => new Date(isoString),
        encode: (date) => date.toISOString(),
    },
);

export const DocumentSchema = z.object({
    id: z.string(),
    title: z.string(),
    content: z.string().check(z.minLength(5), z.trim()),
    created_at: stringToDate,
    updated_at: stringToDate,
});

export type DocumentType = z.infer<typeof DocumentSchema>;
