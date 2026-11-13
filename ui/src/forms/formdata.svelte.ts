// Reads props once, at mount, for a component that is mounted for one value
// (a field of a panel). The thunk keeps the reads inside a closure, as the
// compiler requires for intentional one-time reads.
export function propsOnce<T>(get: () => T): T {
	return get();
}
