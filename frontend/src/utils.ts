export async function safe<T>(
    promise: Promise<T>,
): Promise<[T | null, Error | null]> {
    try {
        return [await promise, null];
    } catch (err) {
        return [null, err as Error];
    }
}
