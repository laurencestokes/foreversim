import type { IMessageType, JsonReadOptions, JsonValue } from '@protobuf-ts/runtime';

// Reads saved or imported JSON, skipping the fields and enum names the protos do not list. With
// ignoreUnknownFields, protobuf-ts picks a oneof's case before it reads the value, so an enum name it
// doesn't know (an APL placeholder the protos do not list, in an ActionID, say) leaves the case set
// with an undefined value, which throws once the message is written to binary. Those oneofs are
// cleared here.
export function fromJsonLenient<T extends object>(type: IMessageType<T>, json: JsonValue, options?: Partial<JsonReadOptions>): T {
	return clearUnreadOneofs(type, type.fromJson(json, { ...options, ignoreUnknownFields: true }));
}

export function fromJsonStringLenient<T extends object>(type: IMessageType<T>, json: string, options?: Partial<JsonReadOptions>): T {
	return fromJsonLenient(type, JSON.parse(json), options);
}

function clearUnreadOneofs<T extends object>(type: IMessageType<T>, message: T): T {
	const fields = message as Record<string, any>;
	for (const field of type.fields) {
		let value;
		if (field.oneof) {
			const group = fields[field.oneof];
			if (group?.oneofKind !== field.localName) continue;
			value = group[field.localName];
			if (value === undefined) {
				fields[field.oneof] = { oneofKind: undefined };
				continue;
			}
		} else {
			value = fields[field.localName];
		}

		if (field.kind === 'message') {
			const subType = field.T();
			if (Array.isArray(value)) value.forEach(item => clearUnreadOneofs(subType, item));
			else if (value) clearUnreadOneofs(subType, value);
		} else if (field.kind === 'map' && field.V.kind === 'message' && value) {
			const subType = field.V.T();
			Object.values(value).forEach(item => clearUnreadOneofs(subType, item as object));
		}
	}
	return message;
}
