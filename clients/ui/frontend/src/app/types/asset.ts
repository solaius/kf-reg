/** Generic entity types for capabilities-driven rendering. */

// A generic entity is a JSON object - fields are accessed via capabilities paths
export type GenericEntity = Record<string, unknown>;

export type GenericEntityList = {
  items: GenericEntity[];
  size: number;
  nextPageToken?: string;
};
