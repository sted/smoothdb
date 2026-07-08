// Seeds a form from its `data` prop, captured once at mount (the forms are
// remounted by ModalPanel on each open). Takes a thunk so the prop is read
// inside a closure, as the compiler requires for intentional one-time reads.
//
// `initial` is a plain snapshot used for dirty-checking; `current` is a
// reactive deep copy the form can freely mutate — editing then cancelling
// never touches the caller's row object.
export function formData<T extends object>(get: () => T): { initial: T; current: T } {
	const initial = $state.snapshot(get()) as T;
	let current = $state($state.snapshot(get()) as T);
	return { initial, current };
}
