/**
 * Domain Entity: Category
 * Represents a junk cleaning category in the system
 */
export class Category {
  constructor(
    public readonly id: string,
    public readonly name: string,
    public readonly description: string,
    public readonly isAggressive: boolean,
    public readonly minAgeHours?: number
  ) {}

  static createSafe(
    id: string,
    name: string,
    description: string,
    minAgeHours?: number
  ): Category {
    return new Category(id, name, description, false, minAgeHours);
  }

  static createAggressive(
    id: string,
    name: string,
    description: string,
    minAgeHours?: number
  ): Category {
    return new Category(id, name, description, true, minAgeHours);
  }

  equals(other: Category): boolean {
    return this.id === other.id;
  }

  toString(): string {
    return `${this.name} (${this.id})`;
  }
}

/**
 * Value Object: CategorySelection
 * Immutable selection state for categories
 */
export class CategorySelection {
  constructor(
    private readonly selectedIds: ReadonlySet<string>
  ) {}

  static empty(): CategorySelection {
    return new CategorySelection(new Set());
  }

  static all(categories: Category[]): CategorySelection {
    return new CategorySelection(new Set(categories.map(c => c.id)));
  }

  select(id: string): CategorySelection {
    const newSet = new Set(this.selectedIds);
    newSet.add(id);
    return new CategorySelection(newSet);
  }

  deselect(id: string): CategorySelection {
    const newSet = new Set(this.selectedIds);
    newSet.delete(id);
    return new CategorySelection(newSet);
  }

  toggle(id: string): CategorySelection {
    if (this.isSelected(id)) {
      return this.deselect(id);
    }
    return this.select(id);
  }

  isSelected(id: string): boolean {
    return this.selectedIds.has(id);
  }

  getSelectedIds(): string[] {
    return Array.from(this.selectedIds);
  }

  get count(): number {
    return this.selectedIds.size;
  }
}
