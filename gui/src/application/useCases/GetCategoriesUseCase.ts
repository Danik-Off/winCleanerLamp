/**
 * Application Use Case: GetCategoriesUseCase
 * Retrieves available cleaning categories
 */
import { Category, CategorySelection } from '@domain/index';
import type { ICleanerService } from '../ports/ICleanerService';

export interface CategoriesResponse {
  safe: Category[];
  aggressive: Category[];
  defaultSelection: CategorySelection;
}

export class GetCategoriesUseCase {
  constructor(
    private readonly cleanerService: ICleanerService
  ) {}

  async execute(): Promise<CategoriesResponse> {
    const { safe, aggressive } = await this.cleanerService.getCategories();
    
    // By default, select all safe categories
    const defaultSelection = CategorySelection.all(safe);

    return {
      safe,
      aggressive,
      defaultSelection
    };
  }
}
